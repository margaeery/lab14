import os

import duckdb
import matplotlib
import matplotlib.pyplot as plt

matplotlib.use("Agg")


def get_parquet_path():
    project_root = os.path.dirname(
        os.path.dirname(os.path.abspath(__file__)),
    )
    return os.path.join(
        project_root, "analytics", "leagues_clean.parquet",
    )


def get_aggregated_data(parquet_path):
    conn = duckdb.connect()
    query = f"""
        SELECT
            country.name AS country_name,
            COUNT(*) AS league_count,
            SUM(id) AS league_id_sum,
            AVG(id) AS league_id_avg
        FROM '{parquet_path}'
        WHERE country.name IS NOT NULL
        GROUP BY country.name
        ORDER BY league_count DESC
    """
    result = conn.execute(query).fetchdf()
    conn.close()
    return result


def plot_top_countries_bar(df, output_path):
    top_n = 15
    df_top = df.head(top_n)

    fig, ax = plt.subplots(figsize=(12, 8))
    bars = ax.barh(
        df_top["country_name"][::-1],
        df_top["league_count"][::-1],
        color="steelblue",
    )
    ax.set_xlabel("Number of Leagues", fontsize=12)
    ax.set_ylabel("Country", fontsize=12)
    ax.set_title(
        f"Top {top_n} Countries by Number of Leagues",
        fontsize=14,
        fontweight="bold",
    )
    ax.grid(axis="x", linestyle="--", alpha=0.7)

    for bar in bars:
        width = bar.get_width()
        ax.text(
            width + 1,
            bar.get_y() + bar.get_height() / 2,
            f"{int(width)}",
            ha="left",
            va="center",
            fontsize=9,
        )

    plt.tight_layout()
    fig.savefig(output_path, dpi=150, bbox_inches="tight")
    plt.close(fig)


def plot_league_distribution_pie(df, output_path):
    top_n = 10
    df_top = df.head(top_n).copy()
    others_count = df.iloc[top_n:]["league_count"].sum()

    labels = list(df_top["country_name"]) + ["Others"]
    sizes = list(df_top["league_count"]) + [others_count]
    colors = plt.cm.Set3(range(len(labels)))

    fig, ax = plt.subplots(figsize=(10, 10))
    wedges, texts, autotexts = ax.pie(
        sizes,
        labels=labels,
        autopct="%1.1f%%",
        startangle=90,
        colors=colors,
        textprops={"fontsize": 10},
    )
    ax.set_title(
        "Distribution of Leagues by Country",
        fontsize=14,
        fontweight="bold",
    )

    for autotext in autotexts:
        autotext.set_color("white")
        autotext.set_fontweight("bold")

    plt.tight_layout()
    fig.savefig(output_path, dpi=150, bbox_inches="tight")
    plt.close(fig)


def main():
    parquet_path = get_parquet_path()
    df = get_aggregated_data(parquet_path)

    project_root = os.path.dirname(
        os.path.dirname(os.path.abspath(__file__)),
    )
    output_dir = os.path.join(project_root, "analytics")
    os.makedirs(output_dir, exist_ok=True)

    bar_path = os.path.join(output_dir, "top_countries_bar.png")
    pie_path = os.path.join(output_dir, "league_distribution_pie.png")

    plot_top_countries_bar(df, bar_path)
    plot_league_distribution_pie(df, pie_path)

    print(f"Bar chart saved to: {bar_path}")
    print(f"Pie chart saved to: {pie_path}")


if __name__ == "__main__":
    main()
