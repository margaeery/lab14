import json

import polars as pl
import pytest

from clean import (
    cast_types,
    fill_nulls,
    fill_struct_nulls,
    load_data,
    remove_duplicates,
    save_to_parquet,
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


def test_save_to_parquet_creates_file(tmp_path):
    output_path = tmp_path / "output.parquet"
    df = pl.DataFrame({
        "id": [1, 2, 3],
        "name": ["A", "B", "C"],
    })

    save_to_parquet(df, str(output_path))

    assert output_path.exists()

    loaded = pl.read_parquet(str(output_path))
    assert loaded.height == 3
    assert loaded.width == 2
    assert loaded["id"].to_list() == [1, 2, 3]
    assert loaded["name"].to_list() == ["A", "B", "C"]


def test_save_to_parquet_empty_frame(tmp_path):
    output_path = tmp_path / "empty.parquet"
    df = pl.DataFrame({
        "id": pl.Series([], dtype=pl.Int64),
        "name": pl.Series([], dtype=pl.String),
    })

    save_to_parquet(df, str(output_path))

    assert output_path.exists()

    loaded = pl.read_parquet(str(output_path))
    assert loaded.height == 0
    assert loaded.width == 2


def test_save_to_parquet_with_struct(tmp_path):
    output_path = tmp_path / "struct.parquet"
    df = pl.DataFrame({
        "id": [1, 2],
        "country": [
            {"id": 10, "code": "AA"},
            {"id": 20, "code": "BB"},
        ],
    })

    save_to_parquet(df, str(output_path))

    loaded = pl.read_parquet(str(output_path))
    assert loaded.height == 2
    assert loaded["country"][0]["id"] == 10
    assert loaded["country"][1]["code"] == "BB"


def test_save_to_parquet_overwrites_existing(tmp_path):
    output_path = tmp_path / "overwrite.parquet"
    df1 = pl.DataFrame({"id": [1, 2]})
    df2 = pl.DataFrame({"id": [3, 4, 5]})

    save_to_parquet(df1, str(output_path))
    save_to_parquet(df2, str(output_path))

    loaded = pl.read_parquet(str(output_path))
    assert loaded.height == 3
    assert loaded["id"].to_list() == [3, 4, 5]


def test_save_to_parquet_invalid_path():
    df = pl.DataFrame({"id": [1]})

    with pytest.raises(Exception):
        save_to_parquet(df, "/nonexistent/directory/file.parquet")


def test_save_to_parquet_preserves_schema(tmp_path):
    output_path = tmp_path / "schema.parquet"
    df = pl.DataFrame({
        "int_col": pl.Series([1, 2], dtype=pl.Int64),
        "float_col": pl.Series([1.5, 2.5], dtype=pl.Float64),
        "str_col": pl.Series(["a", "b"], dtype=pl.String),
        "bool_col": pl.Series([True, False], dtype=pl.Boolean),
    })

    save_to_parquet(df, str(output_path))

    loaded = pl.read_parquet(str(output_path))
    assert loaded["int_col"].dtype == pl.Int64
    assert loaded["float_col"].dtype == pl.Float64
    assert loaded["str_col"].dtype == pl.String
    assert loaded["bool_col"].dtype == pl.Boolean
