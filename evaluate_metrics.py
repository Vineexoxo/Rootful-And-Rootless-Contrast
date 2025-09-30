#!/usr/bin/env python3
"""
Python evaluator for rootless vs rootful container performance.

Behavior:
- Queries Prometheus for a set of container metrics for two roles: rootless and rootful
- Computes per-metric scores using a normalized delta mapped to 0..100 where 50 is neutral
- Aggregates via weighted average to produce an overall score and verdict
- Supports multiple repetitions and outputs mean/stddev across runs
- Treats missing series as N/A (excluded from weighting) to avoid bias
"""
import argparse
import json
import os
import sys
from typing import Dict, Tuple, Optional, List
import time

import requests


def prom_query(prom_url: str, query: str) -> Tuple[Optional[float], bool]:
    """Execute an instant PromQL query and return (value, ok).

    - Returns (None, False) when there are no samples, so callers can exclude
      that metric from scoring for this repetition.
    - Returns (float, True) when a numeric value was parsed successfully.
    """
    r = requests.get(f"{prom_url}/api/v1/query", params={"query": query}, timeout=20)
    r.raise_for_status()
    data = r.json()
    if data.get("status") != "success":
        raise RuntimeError(f"Prometheus query failed: {data}")
    result = data.get("data", {}).get("result", [])
    if not result:
        return None, False
    # use the first sample's value
    value = result[0]["value"][1]
    try:
        return float(value), True
    except Exception:
        return None, False


def compute_score(rootless: float, rootful: float, weight: float, direction: str) -> Tuple[float, float]:
    """Compute normalized delta and weighted score contribution for one metric.

    Steps:
    - Normalized delta rel in [-1, 1] using a symmetric denominator:
      rel = (rootless - rootful) / ((|rootless| + |rootful|) / 2)
      Guarded for zero denominator.
    - If lower-is-better, invert rel.
    - Map rel to a 0..100 score with 50 = neutral: score = 50 + 50*rel
    - Clip to [0, 100] and multiply by the metric weight for contribution.
    """
    # normalized delta in [-1, 1]
    denom = abs(rootless) + abs(rootful)
    if denom == 0:
        rel = 0.0
    else:
        rel = (rootless - rootful) / (denom / 2.0)
        rel = max(-1.0, min(1.0, rel))
    if direction == "lower":
        rel = -rel
    score = 50.0 + (rel * 50.0)
    score = max(0.0, min(100.0, score))
    return rel, score * weight


def main():
    """CLI entrypoint: loads config, runs N repetitions, writes JSON report."""
    parser = argparse.ArgumentParser(description="Evaluate rootless vs rootful metrics from Prometheus")
    parser.add_argument("--config", default="evaluator_config.json", help="Path to evaluator config JSON")
    parser.add_argument("--prom", default=None, help="Override Prometheus URL")
    parser.add_argument("--rootless", default=None, help="Rootless container name")
    parser.add_argument("--rootful", default=None, help="Rootful container name")
    parser.add_argument("--reps", type=int, default=1, help="Number of repetitions for statistics")
    parser.add_argument("--rep-interval", type=float, default=5.0, help="Seconds to wait between repetitions")
    args = parser.parse_args()

    with open(args.config, "r") as f:
        cfg = json.load(f)

    prom_url = args.prom or cfg["prometheus_url"]
    rootless_name = args.rootless or cfg["rootless_container"]
    rootful_name = args.rootful or cfg["rootful_container"]
    metrics: Dict[str, Dict] = cfg["metrics"]
    thresholds = cfg["thresholds"]
    out_path = cfg["output"]["path"]

    # Each entry contains metrics, aggregate score, and verdict for a single repetition
    runs: List[Dict] = []

    def with_container(base: str, container: str) -> str:
        """Add container label match to a base query.

        Handles two cases:
        - base has an existing matcher {..}: inject ",container=\"name\"}" before the closing brace
        - base has no matcher: append {container="name"}
        """
        if "{" in base:
            head, tail = base.rsplit('}', 1)
            return f"{head},container=\"{container}\"}}{tail}"
        return f"{base}{{container=\"{container}\"}}"

    # Repeat to capture variability over time; wait between reps if configured
    for rep in range(args.reps):
        rep_metrics = {}
        total_weight = 0.0
        weighted_sum = 0.0
        for name, spec in metrics.items():
            q = spec["query"]
            weight = float(spec.get("weight", 0.0))
            direction = spec.get("direction", "higher")

            rl_val, rl_ok = prom_query(prom_url, with_container(q, rootless_name))
            rf_val, rf_ok = prom_query(prom_url, with_container(q, rootful_name))

            # Missing series for either role: exclude this metric from the weighting
            if not (rl_ok and rf_ok):
                rep_metrics[name] = {
                    "rootless_value": rl_val,
                    "rootful_value": rf_val,
                    "direction": direction,
                    "weight": weight,
                    "included": False,
                    "reason": "missing series"
                }
                continue

            rel, weighted = compute_score(rl_val or 0.0, rf_val or 0.0, weight, direction)
            rep_metrics[name] = {
                "rootless_value": rl_val,
                "rootful_value": rf_val,
                "direction": direction,
                "weight": weight,
                "included": True,
                "normalized_delta": rel,
                "weighted_score_contrib": weighted,
            }
            total_weight += weight
            weighted_sum += weighted

        # Aggregate into overall weighted score for this repetition
        if total_weight > 0:
            weighted_score = weighted_sum / total_weight
        else:
            weighted_score = 50.0

        # Map overall score to requested verdict thresholds
        if weighted_score >= thresholds["strong_rootless"]:
            verdict = "strong prefer rootless"
        elif weighted_score >= thresholds["mild_rootless"]:
            verdict = "mild prefer rootless"
        elif weighted_score <= thresholds["strong_rootful"]:
            verdict = "strong prefer rootful"
        elif weighted_score <= thresholds["mild_rootful"]:
            verdict = "mild prefer rootful"
        else:
            verdict = "inconclusive"

        runs.append({
            "rep": rep + 1,
            "timestamp": int(time.time()),
            "metrics": rep_metrics,
            "weighted_score": weighted_score,
            "verdict": verdict,
            "effective_weight": total_weight,
        })

        if rep + 1 < args.reps:
            time.sleep(args.rep_interval)

    # Compute mean and stddev across repetitions for stability analysis
    scores = [r["weighted_score"] for r in runs]
    mean_score = sum(scores) / len(scores) if scores else 50.0
    variance = sum((s - mean_score) ** 2 for s in scores) / len(scores) if scores else 0.0
    stddev = variance ** 0.5

    summary = {
        "rootless_target": rootless_name,
        "rootful_target": rootful_name,
        "repetitions": args.reps,
        "mean_weighted_score": mean_score,
        "stddev_weighted_score": stddev,
        "min_weighted_score": min(scores) if scores else 50.0,
        "max_weighted_score": max(scores) if scores else 50.0,
        "thresholds": thresholds,
        "final_verdict": runs[-1]["verdict"] if runs else "inconclusive"
    }

    output = {
        "compared_at": int(time.time()),
        "summary": summary,
        "runs": runs,
    }

    os.makedirs(os.path.dirname(out_path) or ".", exist_ok=True)
    with open(out_path, "w") as f:
        json.dump(output, f, indent=2)

    print(f"Report written to {out_path} (mean={mean_score:.2f} ± {stddev:.2f})")


if __name__ == "__main__":
    main()


