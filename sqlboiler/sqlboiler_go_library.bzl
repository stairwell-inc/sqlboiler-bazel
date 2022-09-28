load(
    "@io_bazel_rules_go//go:def.bzl",
    "GoLibrary",
    "go_context",
)

def _sqlboiler_go_library_impl(ctx):
    output_dir = ctx.actions.declare_directory("models.go")

    args = ctx.actions.args()
    args.add_all([
        "--schema", ctx.attr.schema,
        "--sql", ctx.file.sql.path,
        "--output_dir", output_dir.path,
        "--sqlboiler", ctx.executable._sqlboiler.path,
        "--sqlboiler_psql", ctx.executable._sqlboiler_psql.path,
        "--postgres_bin_dir", ctx.files._postgres_dir[0].path + "/bin",
    ])

    if hasattr(ctx.file.config, "path"): args.add_all(["--config", ctx.file.config.path])
    if len(ctx.attr.create_users) > 0:
        args.add_all([
            "--create_users",
            ",".join(ctx.attr.create_users),
        ])

    input_direct_deps = [ctx.file.sql]
    if ctx.file.config:
        input_direct_deps.append(ctx.file.config)

    ctx.actions.run(
        inputs = depset(
            direct = input_direct_deps + ctx.files._postgres_dir,
        ),
        outputs = [output_dir],
        progress_message = "Running sqlboiler for %s" % ctx.label,
        executable = ctx.executable._run_sqlboiler,
        tools = [ctx.executable._sqlboiler, ctx.executable._sqlboiler_psql],
        arguments = [args],
    )

    go = go_context(ctx)
    library = go.new_library(go, srcs = [output_dir] + ctx.files.srcs)

    source = go.library_to_source(go, struct(
        importpath = ctx.attr.importpath,
        deps = ctx.attr._implicit_deps + ctx.attr.deps,
        _go_context_data = ctx.attr._go_context_data,
    ), library, ctx.coverage_instrumented())

    return [
        library,
        source,
        OutputGroupInfo(go_generated_srcs = [output_dir]),
    ]

sqlboiler_go_library = rule(
    toolchains = ["@io_bazel_rules_go//go:toolchain"],
    attrs = {
        "sql": attr.label(
            allow_single_file = [".sql"],
            mandatory = True,
        ),
        "schema": attr.string(
            mandatory = True,
        ),
        "importpath": attr.string(
            mandatory = True,
        ),
        "create_users": attr.string_list(
            mandatory = False,
            doc = """List of users to create before running.

Some schemas refer to users that should preexist. This allows you to create
those users in the testing postgres before generating the sqlboiler""",
        ),
        "config": attr.label(
            allow_single_file = True,
            mandatory = False,
            doc = "Provide additional sqlboiler configuration (in .toml format)",
        ),
        "srcs": attr.label_list(
            allow_files = True,
            doc = "List of additional files to include in the compilation unit",
        ),
        "deps": attr.label_list(
            doc = "List of additional dependencies to include in the compilation unit",
            providers = [GoLibrary],
        ),
        "_go_context_data": attr.label(default = "@io_bazel_rules_go//:go_context_data"),
        "_implicit_deps": attr.label_list(
            default = [
                "@com_github_friendsofgo_errors//:errors",
                "@com_github_volatiletech_null_v8//:null",
                "@com_github_volatiletech_sqlboiler_v4//boil",
                "@com_github_volatiletech_sqlboiler_v4//drivers",
                "@com_github_volatiletech_sqlboiler_v4//types",
                "@com_github_volatiletech_sqlboiler_v4//queries",
                "@com_github_volatiletech_sqlboiler_v4//queries/qm",
                "@com_github_volatiletech_sqlboiler_v4//queries/qmhelper",
                "@com_github_volatiletech_strmangle//:strmangle",
            ],
            providers = [GoLibrary],
        ),
        "_sqlboiler": attr.label(
            allow_single_file = True,
            cfg = "exec",
            default = Label("@com_github_volatiletech_sqlboiler_v4//:v4"),
            executable = True,
        ),
        "_sqlboiler_psql": attr.label(
            allow_single_file = True,
            cfg = "exec",
            default = Label("@com_github_volatiletech_sqlboiler_v4//drivers/sqlboiler-psql:sqlboiler-psql"),
            executable = True,
        ),
        "_postgres_dir": attr.label(
            allow_files = True,
            default = Label("//psql"),
        ),
        "_run_sqlboiler": attr.label(
            allow_single_file = True,
            cfg = "exec",
            default = Label("//sqlboiler/cmd:sqlboiler"),
            executable = True,
        ),
    },
    implementation = _sqlboiler_go_library_impl,
)
