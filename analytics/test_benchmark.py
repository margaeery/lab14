import os

import polars as pl
import pytest

from benchmark import get_parquet_path, run_duckdb_query


def create_test_parquet(path, data):
    df = pl.DataFrame(data)
    df.write_parquet(path)


def test_get_parquet_path_exists():
    path = get_parquet_path()
    assert os.path.isfile(path)


def test_duckdb_query_returns_correct_columns(tmp_path):
    parquet_path = tmp_path / "test.parquet"
    create_test_parquet(
        parquet_path,
        {
            "id": [1, 2],
            "name": ["League A", "League B"],
            "url": ["", ""],
            "country": [
                {"id": 10, "code": "AA", "name": "Aland", "url": ""},
                {"id": 11, "code": "BB", "name": "Aland", "url": ""},
            ],
        },
    )
    result = run_duckdb_query(str(parquet_path))
    expected_columns = {
        "country_name",
        "league_count",
        "league_id_sum",
        "league_id_avg",
        "league_id_min",
        "league_id_max",
    }
    assert set(result.columns) == expected_columns


def test_duckdb_query_filters_null_country_name(tmp_path):
    parquet_path = tmp_path / "test.parquet"
    create_test_parquet(
        parquet_path,
        {
            "id": [1, 2, 3],
            "name": ["League A", "League B", "League C"],
            "url": ["", "", ""],
            "country": [
                {"id": 10, "code": "AA", "name": "Aland", "url": ""},
                {"id": 11, "code": "BB", "name": None, "url": ""},
                {"id": 12, "code": "CC", "name": "Aland", "url": ""},
            ],
        },
    )
    result = run_duckdb_query(str(parquet_path))
    assert result.height == 1
    row = result.row(0, named=True)
    assert row["country_name"] == "Aland"
    assert row["league_count"] == 2


def test_duckdb_query_groups_and_aggregates(tmp_path):
    parquet_path = tmp_path / "test.parquet"
    create_test_parquet(
        parquet_path,
        {
            "id": [10, 20, 30, 40],
            "name": ["A", "B", "C", "D"],
            "url": ["", "", "", ""],
            "country": [
                {"id": 1, "code": "AA", "name": "Aland", "url": ""},
                {"id": 1, "code": "AA", "name": "Aland", "url": ""},
                {"id": 2, "code": "BB", "name": "Bland", "url": ""},
                {"id": 2, "code": "BB", "name": "Bland", "url": ""},
            ],
        },
    )
    result = run_duckdb_query(str(parquet_path))
    assert result.height == 2
    aland = result.filter(pl.col("country_name") == "Aland").row(0, named=True)
    assert aland["league_count"] == 2
    assert aland["league_id_sum"] == 30
    assert aland["league_id_avg"] == 15.0
    assert aland["league_id_min"] == 10
    assert aland["league_id_max"] == 20


def test_duckdb_query_having_filter_excludes_single_countries(tmp_path):
    parquet_path = tmp_path / "test.parquet"
    create_test_parquet(
        parquet_path,
        {
            "id": [1, 2, 3],
            "name": ["A", "B", "C"],
            "url": ["", "", ""],
            "country": [
                {"id": 1, "code": "AA", "name": "Aland", "url": ""},
                {"id": 1, "code": "AA", "name": "Aland", "url": ""},
                {"id": 2, "code": "BB", "name": "Bland", "url": ""},
            ],
        },
    )
    result = run_duckdb_query(str(parquet_path))
    assert result.height == 1
    assert result["country_name"][0] == "Aland"


def test_duckdb_query_sorts_descending(tmp_path):
    parquet_path = tmp_path / "test.parquet"
    create_test_parquet(
        parquet_path,
        {
            "id": [1, 2, 3, 4, 5],
            "name": ["A", "B", "C", "D", "E"],
            "url": ["", "", "", "", ""],
            "country": [
                {"id": 1, "code": "AA", "name": "Aland", "url": ""},
                {"id": 1, "code": "AA", "name": "Aland", "url": ""},
                {"id": 1, "code": "AA", "name": "Aland", "url": ""},
                {"id": 2, "code": "BB", "name": "Bland", "url": ""},
                {"id": 2, "code": "BB", "name": "Bland", "url": ""},
            ],
        },
    )
    result = run_duckdb_query(str(parquet_path))
    counts = result["league_count"].to_list()
    assert counts == sorted(counts, reverse=True)
    assert counts[0] == 3
    assert counts[1] == 2


def test_duckdb_query_missing_file():
    with pytest.raises(Exception):
        run_duckdb_query("nonexistent_file.parquet")


def test_duckdb_query_invalid_file(tmp_path):
    invalid_path = tmp_path / "invalid.txt"
    invalid_path.write_text("not a parquet file")
    with pytest.raises(Exception):
        run_duckdb_query(str(invalid_path))


def test_duckdb_query_empty_result(tmp_path):
    parquet_path = tmp_path / "test.parquet"
    create_test_parquet(
        parquet_path,
        {
            "id": [1],
            "name": ["A"],
            "url": [""],
            "country": [
                {"id": 1, "code": "AA", "name": "Aland", "url": ""},
            ],
        },
    )
    result = run_duckdb_query(str(parquet_path))
    assert result.height == 0
