load("@rules_foreign_cc//foreign_cc:defs.bzl", "configure_make")

exports_files(["LICENSE"])

# NOTE: This target is not hermetic.
# Host system must meet requirements outlined in: https://wiki.postgresql.org/wiki/Compile_and_Install_from_source_code
configure_make(
    name = "psql_compile",
    configure_options = [
        "--without-zlib",
        "--without-readline",
    ],
    lib_source = "@postgresql//:all",
    out_headers_only = True,
)

filegroup(
    name = "psql",
    srcs = [
        ":psql_compile",
    ],
    output_group = "gen_dir",
    visibility = [
        "//visibility:public",
    ],
)
