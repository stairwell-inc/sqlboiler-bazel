load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "cmd_lib",
    srcs = [
        "main.go",
    ],
    importpath = "github.com/mrmeku/sqlboiler-bazel/sqlboiler/cmd",
    visibility = ["//visibility:private"],
    deps = [
        "//psqltesting",
        "@com_github_pelletier_go_toml//:go-toml",
    ],
)

go_binary(
    name = "sqlboiler",
    embed = [":cmd_lib"],
    visibility = ["//visibility:public"],
)
