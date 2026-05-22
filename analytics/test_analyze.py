import json
import os

import polars as pl
import pytest

from analyze import load_data, get_data_info

def test_load_data_valid_file(tmp_path):
    file_path = tmp_path / "valid.json"
    records = [
        {"id": 1, "name": "League A", "url": "", "country": {"id": 10, "code": "AA", "name": "Aland", "url": ""}},
        {"id": 2, "name": "League B", "url": "", "country": {"id": 11, "code": "BB", "name": "Bland", "url": ""}},
    ]
    with open(file_path, "w", encoding="utf-8") as f:
        for record in records:
            f.write(json.dumps(record) + "\n")

    df = load_data(str(file_path))

    assert df.height == 2
    assert df.width == 4
    assert df.columns == ["id", "name", "url", "country"]

def test_get_data_info():
    df = pl.DataFrame({
        "id": [1, 2, 3],
        "name": ["A", "B", "C"],
    })

    info = get_data_info(df)

    assert info["rows"] == 3
    assert info["columns"] == 2
    assert info["head"].height == 3
    assert info["schema"] == {"id": pl.Int64, "name": pl.String}
    assert info["null_counts"]["id"][0] == 0
    assert info["null_counts"]["name"][0] == 0

def test_load_data_file_not_found():
    with pytest.raises(Exception):
        load_data("nonexistent_file_xyz.json")

def test_load_data_invalid_json(tmp_path):
    file_path = tmp_path / "invalid.json"
    with open(file_path, "w", encoding="utf-8") as f:
        f.write("not a json line\n")

    with pytest.raises(Exception):
        load_data(str(file_path))
