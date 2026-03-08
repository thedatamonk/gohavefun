#!/usr/bin/env python3
"""Fetch features from Go feature store, generate churn labels, train XGBoost."""

import json
import os
import requests
import pandas as pd
import xgboost as xgb
from sklearn.model_selection import train_test_split
from sklearn.metrics import accuracy_score, classification_report

BASE_URL = os.environ.get("FEATURE_STORE_URL", "http://localhost:8080")
MODEL_DIR = os.path.join(os.path.dirname(__file__), "..", "models")

FEATURE_NAMES = [
    "tenure_months", "plan_tier", "monthly_charge",
    "logins_last_30d", "avg_session_minutes", "days_since_last_login", "feature_adoption_pct",
    "total_spend", "late_payments_count", "avg_monthly_spend",
    "tickets_last_90d", "avg_resolution_hours", "escalation_count",
]


def fetch_all_features(n_customers=5000):
    """Fetch features for all customers from the Go feature store."""
    rows = []
    for i in range(1, n_customers + 1):
        cust_id = f"cust-{i:04d}"
        resp = requests.get(f"{BASE_URL}/customers/{cust_id}/features")
        if resp.status_code != 200:
            print(f"Warning: {cust_id} returned {resp.status_code}, skipping")
            continue

        data = resp.json()
        row = {"customer_id": cust_id}
        for group_name, features in data.items():
            if isinstance(features, dict):
                row.update(features)
        rows.append(row)

        if i % 500 == 0:
            print(f"Fetched {i}/{n_customers} customers")

    return pd.DataFrame(rows)


def generate_labels(df):
    """Generate synthetic churn labels from feature heuristics."""
    churned = (
        ((df["days_since_last_login"] > 15) & (df["escalation_count"] >= 2))
        | ((df["logins_last_30d"] < 3) & (df["late_payments_count"] >= 3))
    )
    df["churned"] = churned.astype(int)
    return df


def train_model(df):
    """Train XGBoost binary classifier."""
    X = df[FEATURE_NAMES]
    y = df["churned"]

    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=0.2, random_state=42, stratify=y
    )

    model = xgb.XGBClassifier(
        n_estimators=100,
        max_depth=4,
        learning_rate=0.1,
        objective="binary:logistic",
        eval_metric="logloss",
        random_state=42,
    )
    model.fit(X_train, y_train, eval_set=[(X_test, y_test)], verbose=True)

    # Evaluate
    y_pred = model.predict(X_test)
    print(f"\nAccuracy: {accuracy_score(y_test, y_pred):.4f}")
    print(classification_report(y_test, y_pred, target_names=["retained", "churned"]))

    return model


def export_model(model):
    """Export XGBoost model as JSON for Go inference."""
    os.makedirs(MODEL_DIR, exist_ok=True)
    model_path = os.path.join(MODEL_DIR, "model.json")
    model.save_model(model_path)
    print(f"Model saved to {model_path}")

    # Also save feature names for Go to use
    meta = {"feature_names": FEATURE_NAMES, "n_estimators": model.n_estimators}
    meta_path = os.path.join(MODEL_DIR, "meta.json")
    with open(meta_path, "w") as f:
        json.dump(meta, f, indent=2)
    print(f"Metadata saved to {meta_path}")


def main():
    print("=== Fetching features from Go feature store ===")
    df = fetch_all_features()
    print(f"Fetched {len(df)} customers, {len(df.columns)} columns")

    print("\n=== Generating churn labels ===")
    df = generate_labels(df)
    print(f"Churn distribution:\n{df['churned'].value_counts()}")

    print("\n=== Training XGBoost ===")
    model = train_model(df)

    print("\n=== Exporting model ===")
    export_model(model)


if __name__ == "__main__":
    main()
