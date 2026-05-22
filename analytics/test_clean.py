import json

import polars as pl
import pytest

from clean import (
    cast_types,
    fill_nulls,
    fill_struct_nulls,
    load_data,
    remove_duplicates,
)


def test_load_data_valid_file(tmp_path):
    file_path = tmp_path / "data.json"
    records = [
        {"id": 1, "name": "A"},
        {"id": 2, "name": "B"},
    ]
    with open(file_path, "w", encoding="utf-8") as f:
        for r in records:
            f.write(json.dumps(r) + "\n")

    df = load_data(str(file_path))

    assert df.height == 2
    assert df.width == 2
    assert df.columns == ["id", "name"]


def test_remove_duplicates_removes_rows():
    df = pl.DataFrame({
        "id": [1, 2, 2, 3],
        "name": ["A", "B", "B", "C"],
    })

    result = remove_duplicates(df)

    assert result.height == 3


def test_remove_duplicates_no_duplicates():
    df = pl.DataFrame({
        "id": [1, 2, 3],
        "name": ["A", "B", "C"],
    })

    result = remove_duplicates(df)

    assert result.height == 3


def test_fill_nulls_string_column():
    df = pl.DataFrame({
        "name": ["A", None, "C"],
    })

    result = fill_nulls(df)

    assert result["name"][1] == "Unknown"


def test_fill_nulls_int_column():
    df = pl.DataFrame({
        "id": [1, None, 3],
    })

    result = fill_nulls(df)

    assert result["id"][1] == 0


def test_fill_nulls_bool_column():
    df = pl.DataFrame({
        "active": [True, None, False],
    })

    result = fill_nulls(df)

    assert result["active"][1] is False


def test_fill_nulls_float_column():
    df = pl.DataFrame({
        "score": [1.5, None, 3.0],
    })

    result = fill_nulls(df)

    assert result["score"][1] == 0.0


def test_fill_nulls_no_nulls():
    df = pl.DataFrame({
        "id": [1, 2, 3],
        "name": ["A", "B", "C"],
    })

    result = fill_nulls(df)

    assert result.height == 3
    assert result.null_count().row(0) == (0, 0)


def test_fill_struct_nulls_fills_missing():
    df = pl.DataFrame({
        "country": [
            {"id": 1, "code": "AA"},
            {"id": None, "code": None},
            {"id": 3, "code": "CC"},
        ],
    })

    result = fill_struct_nulls(df, "country")

    assert result["country"][1]["id"] == 0
    assert result["country"][1]["code"] == "Unknown"


def test_fill_struct_nulls_no_nulls():
    df = pl.DataFrame({
        "country": [
            {"id": 1, "code": "AA"},
            {"id": 2, "code": "BB"},
        ],
    })

    result = fill_struct_nulls(df, "country")

    assert result["country"][0]["id"] == 1
    assert result["country"][0]["code"] == "AA"


def test_cast_types_int32_to_int64():
    df = pl.DataFrame({
        "id": pl.Series([1, 2, 3], dtype=pl.Int32),
    })

    result = cast_types(df)

    assert result["id"].dtype == pl.Int64


def test_cast_types_float64_to_int64():
    df = pl.DataFrame({
        "value": pl.Series([1.0, 2.0, 3.0], dtype=pl.Float64),
    })

    result = cast_types(df)

    assert result["value"].dtype == pl.Int64


def test_cast_types_no_change():
    df = pl.DataFrame({
        "id": pl.Series([1, 2, 3], dtype=pl.Int64),
        "name": pl.Series(["A", "B", "C"], dtype=pl.String),
    })

    result = cast_types(df)

    assert result["id"].dtype == pl.Int64
    assert result["name"].dtype == pl.String


def test_load_data_file_not_found():
    with pytest.raises(Exception):
        load_data("nonexistent_file.json")


def test_load_data_invalid_json(tmp_path):
    file_path = tmp_path / "invalid.json"
    with open(file_path, "w", encoding="utf-8") as f:
        f.write("not json\n")

    with pytest.raises(Exception):
        load_data(str(file_path))


def test_remove_duplicates_empty_frame():
    df = pl.DataFrame({
        "id": [],
        "name": [],
    })

    result = remove_duplicates(df)

    assert result.height == 0


def test_fill_nulls_empty_frame():
    df = pl.DataFrame({
        "id": pl.Series([], dtype=pl.Int64),
        "name": pl.Series([], dtype=pl.String),
    })

    result = fill_nulls(df)

    assert result.height == 0
