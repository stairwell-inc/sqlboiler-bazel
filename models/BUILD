load("//sqlboiler:sqlboiler_go_library.bzl", "sqlboiler_go_library")

sqlboiler_go_library(
    name = "models",
    create_users = [
        "operator",
    ],
    importpath = "github.com/mrmeku/sqlboiler-bazel/models",
    schema = "powerplant",
    sql = "//:schema.sql",
)
