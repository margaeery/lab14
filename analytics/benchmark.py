import os
import time

import duckdb
import polars as pl


def get_parquet_path():
    project_root = os.path.dirname(
        os.path.dirname(os.path.abspath(__file__)),
    )
    return os.path.join(
        project_root, "analytics", "leagues_clean.parquet",
    )


def run_duckdb_query(parquet_path):
    conn = duckdb.connect()
    query = f"""
        SELECT
            country.name AS country_name,
            COUNT(*) AS league_count,
            SUM(id) AS league_id_sum,
            AVG(id) AS league_id_avg,
            MIN(id) AS league_id_min,
            MAX(id) AS league_id_max
        FROM '{parquet_path}'
        WHERE country.name IS NOT NULL
        GROUP BY country.name
        HAVING COUNT(*) > 1
        ORDER BY league_count DESC
    """
    result = conn.execute(query).pl()
    conn.close()
    return result


def run_polars_query(parquet_path):
    df = pl.read_parquet(parquet_path)
    return (
        df.filter(
            pl.col("country").struct.field("name").is_not_null(),
        )
        .group_by(
            pl.col("country").struct.field("name").alias("country_name"),
        )
        .agg(
            pl.col("id").count().alias("league_count"),
            pl.col("id").sum().alias("league_id_sum"),
            pl.col("id").mean().alias("league_id_avg"),
            pl.col("id").min().alias("league_id_min"),
            pl.col("id").max().alias("league_id_max"),
        )
        .filter(pl.col("league_count") > 1)
        .sort("league_count", descending=True)
    )


def benchmark():
    parquet_path = get_parquet_path()

    duckdb_start = time.perf_counter()
    duckdb_result = run_duckdb_query(parquet_path)
    duckdb_elapsed = time.perf_counter() - duckdb_start

    polars_start = time.perf_counter()
    polars_result = run_polars_query(parquet_path)
    polars_elapsed = time.perf_counter() - polars_start

    print("=== DuckDB Result ===")
    print(duckdb_result)
    print()

    print("=== Polars Result ===")
    print(polars_result)
    print()

    print("=== Performance Comparison ===")
    print(f"DuckDB elapsed: {duckdb_elapsed:.6f} s")
    print(f"Polars elapsed: {polars_elapsed:.6f} s")

    if duckdb_elapsed < polars_elapsed:
        faster = "DuckDB"
        ratio = polars_elapsed / duckdb_elapsed
    else:
        faster = "Polars"
        ratio = duckdb_elapsed / polars_elapsed

    print(f"Faster engine: {faster} ({ratio:.2f}x)")


if __name__ == "__main__":
    benchmark()
