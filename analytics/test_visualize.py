import os

import matplotlib
import polars as pl

from visualize import (
    get_aggregated_data,
    plot_league_distribution_pie,
    plot_top_countries_bar,
)


matplotlib.use("Agg")


def create_test_parquet(path):
    df = pl.DataFrame(
        {
            "id": [10, 20, 30, 40, 50],
            "name": ["A", "B", "C", "D", "E"],
            "url": ["", "", "", "", ""],
            "country": [
                {"id": 1, "code": "AA", "name": "Aland", "url": ""},
                {"id": 1, "code": "AA", "name": "Aland", "url": ""},
                {"id": 2, "code": "BB", "name": "Bland", "url": ""},
                {"id": 2, "code": "BB", "name": "Bland", "url": ""},
                {"id": 3, "code": "CC", "name": "Cland", "url": ""},
            ],
        },
    )
    df.write_parquet(path)


def test_get_aggregated_data_returns_expected_columns(tmp_path):
    parquet_path = tmp_path / "leagues.parquet"
    create_test_parquet(parquet_path)

    result = get_aggregated_data(str(parquet_path))

    assert result.shape[0] == 3
    assert list(result.columns) == [
        "country_name",
        "league_count",
        "league_id_sum",
        "league_id_avg",
    ]


def test_plot_top_countries_bar_creates_file(tmp_path):
    output_path = tmp_path / "bar.png"
    df = pl.DataFrame(
        {
            "country_name": ["Aland", "Bland", "Cland"],
            "league_count": [2, 2, 1],
        },
    ).to_pandas()

    plot_top_countries_bar(df, output_path)

    assert output_path.exists()
    assert os.path.getsize(output_path) > 0


def test_plot_league_distribution_pie_creates_file(tmp_path):
    output_path = tmp_path / "pie.png"
    df = pl.DataFrame(
        {
            "country_name": ["Aland", "Bland", "Cland"],
            "league_count": [2, 2, 1],
        },
    ).to_pandas()

    plot_league_distribution_pie(df, output_path)

    assert output_path.exists()
    assert os.path.getsize(output_path) > 0