import logging
import os

import polars as pl


logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s",
)


def load_data(file_path):
    logging.info("Loading data from %s", file_path)
    df = pl.read_ndjson(file_path)
    logging.info("Loaded %d rows and %d columns", df.height, df.width)
    logging.info("Initial schema: %s", df.schema)
    return df


def remove_duplicates(df):
    logging.info("Checking for duplicate rows")
    before = df.height
    df = df.unique()
    after = df.height
    logging.info("Removed %d duplicates, %d rows remaining", before - after, after)
    return df


def fill_nulls(df):
    logging.info("Processing null values")

    for col_name in df.columns:
        dtype = df[col_name].dtype

        if isinstance(dtype, pl.Struct):
            logging.info("Processing struct column '%s'", col_name)
            df = fill_struct_nulls(df, col_name)
        elif dtype in (pl.String, pl.Utf8):
            nulls = df[col_name].null_count()
            if nulls > 0:
                logging.info(
                    "Filling %d nulls in column '%s' with 'Unknown'",
                    nulls, col_name,
                )
                df = df.with_columns(
                    pl.col(col_name).fill_null("Unknown"),
                )
        elif dtype in (
            pl.Int8, pl.Int16, pl.Int32, pl.Int64,
            pl.Float32, pl.Float64,
        ):
            nulls = df[col_name].null_count()
            if nulls > 0:
                logging.info(
                    "Filling %d nulls in column '%s' with 0",
                    nulls, col_name,
                )
                df = df.with_columns(
                    pl.col(col_name).fill_null(0),
                )
        elif dtype == pl.Boolean:
            nulls = df[col_name].null_count()
            if nulls > 0:
                logging.info(
                    "Filling %d nulls in column '%s' with False",
                    nulls, col_name,
                )
                df = df.with_columns(
                    pl.col(col_name).fill_null(False),
                )

    return df


def fill_struct_nulls(df, struct_col):
    fields = df[struct_col].dtype.fields
    field_names = [f.name for f in fields]
    temp_names = [f"__tmp_{struct_col}_{f}" for f in field_names]

    exprs = []
    for field, temp in zip(fields, temp_names):
        field_dtype = field.dtype
        col = pl.col(struct_col).struct.field(field.name)

        if field_dtype in (pl.String, pl.Utf8):
            col = col.fill_null("Unknown")
        elif field_dtype in (
            pl.Int8, pl.Int16, pl.Int32, pl.Int64,
            pl.Float32, pl.Float64,
        ):
            col = col.fill_null(0)
        elif field_dtype == pl.Boolean:
            col = col.fill_null(False)

        exprs.append(col.alias(temp))

    df = df.with_columns(*exprs)

    struct_expr = pl.struct([
        pl.col(tmp).alias(fld) for tmp, fld in zip(temp_names, field_names)
    ]).alias(struct_col)
    df = df.with_columns(struct_expr)
    df = df.drop(temp_names)

    return df


def cast_types(df):
    logging.info("Casting column types")

    for col_name in df.columns:
        dtype = df[col_name].dtype

        if dtype == pl.Int32:
            logging.info("Casting column '%s' from Int32 to Int64", col_name)
            df = df.with_columns(
                pl.col(col_name).cast(pl.Int64),
            )
        elif dtype == pl.Float64:
            logging.info("Casting column '%s' from Float64 to Int64", col_name)
            df = df.with_columns(
                pl.col(col_name).cast(pl.Int64),
            )

    return df


def save_to_parquet(df, file_path):
    logging.info("Saving DataFrame to Parquet: %s", file_path)
    df.write_parquet(file_path)
    logging.info("Saved %d rows to %s", df.height, file_path)


def main():
    project_root = os.path.dirname(
        os.path.dirname(os.path.abspath(__file__)),
    )
    input_path = os.path.join(
        project_root, "collector", "leagues_raw.json",
    )
    output_path = os.path.join(
        project_root, "analytics", "leagues_clean.parquet",
    )

    df = load_data(input_path)
    df = remove_duplicates(df)
    df = fill_nulls(df)
    df = cast_types(df)

    save_to_parquet(df, output_path)

    logging.info("Final dataset: %d rows, %d columns", df.height, df.width)
    logging.info("Final schema: %s", df.schema)


if __name__ == "__main__":
    main()
