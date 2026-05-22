import os

import polars as pl


def load_data(file_path):
    if file_path.lower().endswith(".parquet"):
        return pl.read_parquet(file_path)
    return pl.read_ndjson(file_path)


def extract_country_fields(df):
    return df.with_columns(
        pl.col("country").struct.field("id").alias("country_id"),
        pl.col("country").struct.field("name").alias("country_name"),
    )


def aggregate_by_country_name(df):
    return (
        df.group_by("country_name")
        .agg(
            pl.col("id").count().alias("league_count"),
            pl.col("id").sum().alias("league_id_sum"),
            pl.col("id").mean().alias("league_id_avg"),
            pl.col("id").min().alias("league_id_min"),
            pl.col("id").max().alias("league_id_max"),
            pl.col("country_id").sum().alias("country_id_sum"),
            pl.col("country_id").mean().alias("country_id_avg"),
            pl.col("country_id").min().alias("country_id_min"),
            pl.col("country_id").max().alias("country_id_max"),
        )
        .sort("league_count", descending=True)
    )


def aggregate_by_country_id(df):
    return (
        df.group_by("country_id")
        .agg(
            pl.col("country_name").first().alias("country_name"),
            pl.col("id").count().alias("league_count"),
            pl.col("id").sum().alias("league_id_sum"),
            pl.col("id").mean().alias("league_id_avg"),
            pl.col("id").min().alias("league_id_min"),
            pl.col("id").max().alias("league_id_max"),
        )
        .sort("league_count", descending=True)
    )


def main():
    project_root = os.path.dirname(
        os.path.dirname(os.path.abspath(__file__)),
    )
    file_path = os.path.join(
        project_root, "analytics", "leagues_clean.parquet",
    )

    df = load_data(file_path)
    df = extract_country_fields(df)

    by_name = aggregate_by_country_name(df)
    print("=== Aggregation by Country Name ===")
    print(by_name)
    print()

    by_id = aggregate_by_country_id(df)
    print("=== Aggregation by Country ID ===")
    print(by_id)


if __name__ == "__main__":
    main()