import json

import polars as pl
import pytest

from aggregate import (
    aggregate_by_country_id,
    aggregate_by_country_name,
    extract_country_fields,
    load_data,
)


def test_load_data_valid_file(tmp_path):
    file_path = tmp_path / "data.json"
    records = [
        {
            "id": 1,
            "name": "League A",
            "url": "",
            "country": {"id": 10, "code": "AA", "name": "Aland", "url": ""},
        },
        {
            "id": 2,
            "name": "League B",
            "url": "",
            "country": {"id": 11, "code": "BB", "name": "Bland", "url": ""},
        },
    ]
    with open(file_path, "w", encoding="utf-8") as f:
        for r in records:
            f.write(json.dumps(r) + "\n")

    df = load_data(str(file_path))

    assert df.height == 2
    assert df.width == 4
    assert "country" in df.columns


def test_extract_country_fields():
    df = pl.DataFrame({
        "id": [1, 2],
        "country": [
            {"id": 10, "name": "Aland"},
            {"id": 11, "name": "Bland"},
        ],
    })

    result = extract_country_fields(df)

    assert "country_id" in result.columns
    assert "country_name" in result.columns
    assert result["country_id"].to_list() == [10, 11]
    assert result["country_name"].to_list() == ["Aland", "Bland"]


def test_aggregate_by_country_name():
    df = pl.DataFrame({
        "id": [1, 2, 3, 4],
        "country_id": [10, 10, 20, 20],
        "country_name": ["Aland", "Aland", "Bland", "Bland"],
    })

    result = aggregate_by_country_name(df)

    assert result.height == 2
    row_aland = result.filter(pl.col("country_name") == "Aland").row(0, named=True)
    assert row_aland["league_count"] == 2
    assert row_aland["league_id_sum"] == 3
    assert row_aland["league_id_min"] == 1
    assert row_aland["league_id_max"] == 2
    assert row_aland["country_id_sum"] == 20

    row_bland = result.filter(pl.col("country_name") == "Bland").row(0, named=True)
    assert row_bland["league_count"] == 2
    assert row_bland["league_id_sum"] == 7


def test_aggregate_by_country_id():
    df = pl.DataFrame({
        "id": [1, 2, 3, 4],
        "country_id": [10, 10, 20, 20],
        "country_name": ["Aland", "Aland", "Bland", "Bland"],
    })

    result = aggregate_by_country_id(df)

    assert result.height == 2
    row_10 = result.filter(pl.col("country_id") == 10).row(0, named=True)
    assert row_10["league_count"] == 2
    assert row_10["league_id_sum"] == 3
    assert row_10["country_name"] == "Aland"

    row_20 = result.filter(pl.col("country_id") == 20).row(0, named=True)
    assert row_20["league_count"] == 2
    assert row_20["league_id_sum"] == 7
    assert row_20["country_name"] == "Bland"


def test_aggregate_sorts_descending():
    df = pl.DataFrame({
        "id": [1, 2, 3, 4, 5],
        "country_id": [10, 10, 20, 20, 20],
        "country_name": ["Aland", "Aland", "Bland", "Bland", "Bland"],
    })

    result_name = aggregate_by_country_name(df)
    counts = result_name["league_count"].to_list()
    assert counts == sorted(counts, reverse=True)

    result_id = aggregate_by_country_id(df)
    counts_id = result_id["league_count"].to_list()
    assert counts_id == sorted(counts_id, reverse=True)


def test_load_data_file_not_found():
    with pytest.raises(Exception):
        load_data("nonexistent_file.json")


def test_load_data_invalid_json(tmp_path):
    file_path = tmp_path / "invalid.json"
    with open(file_path, "w", encoding="utf-8") as f:
        f.write("not json\n")

    with pytest.raises(Exception):
        load_data(str(file_path))


def test_extract_country_fields_missing_column():
    df = pl.DataFrame({
        "id": [1, 2],
    })

    with pytest.raises(Exception):
        extract_country_fields(df)


def test_aggregate_by_country_name_empty():
    df = pl.DataFrame({
        "id": pl.Series([], dtype=pl.Int64),
        "country_id": pl.Series([], dtype=pl.Int64),
        "country_name": pl.Series([], dtype=pl.String),
    })

    result = aggregate_by_country_name(df)

    assert result.height == 0


def test_aggregate_by_country_id_empty():
    df = pl.DataFrame({
        "id": pl.Series([], dtype=pl.Int64),
        "country_id": pl.Series([], dtype=pl.Int64),
        "country_name": pl.Series([], dtype=pl.String),
    })

    result = aggregate_by_country_id(df)

    assert result.height == 0
