load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "psqltesting",
    srcs = [
        "postgres.go",
        "safebuf.go",
    ],
    data = ["//psql"],
    importpath = "github.com/mrmeku/sqlboiler-bazel/psqltesting",
    visibility = ["//visibility:public"],
    deps = [
        "@com_github_jackc_pgx_v4//stdlib",
        "@io_bazel_rules_go//go/tools/bazel:go_default_library",
    ],
)
