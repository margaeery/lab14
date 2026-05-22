import polars as pl
import os


def load_data(file_path):
    return pl.read_ndjson(file_path)


def get_data_info(df):
    return {
        "head": df.head(5),
        "schema": df.schema,
        "rows": df.height,
        "columns": df.width,
        "null_counts": df.null_count(),
    }


def main():
    project_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    file_path = os.path.join(project_root, "collector", "leagues_raw.json")

    df = load_data(file_path)
    info = get_data_info(df)

    print("=== First 5 rows ===")
    print(info["head"])
    print()

    print("=== Schema ===")
    print(info["schema"])
    print()

    print("=== Shape ===")
    print(f"Rows: {info['rows']}")
    print(f"Columns: {info['columns']}")
    print()

    print("=== Null counts ===")
    print(info["null_counts"])


if __name__ == "__main__":
    main()