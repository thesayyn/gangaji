# Custom rules and macros to generate Starlark CPU activity

def _char_value(c):
    """Get numeric value for a character."""
    chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.-/"
    idx = chars.find(c)
    return idx if idx >= 0 else 0

def _compute_hash(s):
    """Simple hash computation to burn CPU cycles."""
    h = 0
    for c in s.elems():
        h = (h * 31 + _char_value(c)) % 2147483647
    return h

def _sum(items):
    """Sum a list of numbers."""
    total = 0
    for item in items:
        total += item
    return total

def _generate_content(name, count):
    """Generate content by iterating."""
    result = []
    for i in range(count):
        item = "{}_item_{}".format(name, i)
        h = _compute_hash(item * 10)  # More string processing
        for j in range(20):  # Extra iterations
            h = (h * 31 + j) % 2147483647
        result.append("// {} hash={}".format(item, h))
    return "\n".join(result)

def _heavy_rule_impl(ctx):
    """Rule implementation with heavy Starlark computation."""
    # Do lots of string processing
    content_parts = []
    for i in range(200):
        part = _generate_content(ctx.attr.name, 100)
        content_parts.append(part)

    # Process dependencies
    dep_info = []
    for dep in ctx.attr.deps:
        files = dep[DefaultInfo].files.to_list()
        for f in files:
            h = _compute_hash(f.path)
            dep_info.append("// dep: {} hash={}".format(f.path, h))

    content = "\n".join(content_parts + dep_info)

    out = ctx.actions.declare_file(ctx.attr.name + ".generated.h")
    ctx.actions.write(out, content)

    return [DefaultInfo(files = depset([out]))]

heavy_rule = rule(
    implementation = _heavy_rule_impl,
    attrs = {
        "deps": attr.label_list(allow_files = True),
    },
)

def _analyze_sources_impl(ctx):
    """Analyze source files with heavy computation."""
    analysis = []

    for src in ctx.files.srcs:
        # Compute various metrics
        path_hash = _compute_hash(src.path)
        name_hash = _compute_hash(src.basename)
        ext_hash = _compute_hash(src.extension)

        # Generate analysis report
        report = """
// Analysis for: {path}
// Path hash: {path_hash}
// Name hash: {name_hash}
// Extension hash: {ext_hash}
// Metrics computed: {metrics}
""".format(
            path = src.path,
            path_hash = path_hash,
            name_hash = name_hash,
            ext_hash = ext_hash,
            metrics = path_hash + name_hash + ext_hash,
        )
        analysis.append(report)

    out = ctx.actions.declare_file(ctx.attr.name + ".analysis.txt")
    ctx.actions.write(out, "\n".join(analysis))

    return [DefaultInfo(files = depset([out]))]

analyze_sources = rule(
    implementation = _analyze_sources_impl,
    attrs = {
        "srcs": attr.label_list(allow_files = True),
    },
)

def _matrix_computation(size):
    """Simulate matrix operations."""
    result = []
    for i in range(size):
        row = []
        for j in range(size):
            val = (i * size + j) * 17 % 1000
            # Extra computation
            for k in range(10):
                val = (val * 31 + k) % 10000
            row.append(val)
        result.append(row)

    # Compute "determinant-like" value
    total = 0
    for i in range(size):
        for j in range(size):
            total += result[i][j] * (i + 1) * (j + 1)
    return total

def heavy_genrule(name, **kwargs):
    """Macro that does heavy computation before creating genrule."""
    # Burn CPU cycles with computation
    computations = []
    for i in range(20):
        matrix_val = _matrix_computation(30)
        hash_val = _compute_hash("{}_{}".format(name, i))
        computations.append(matrix_val + hash_val)

    # Generate dynamic content based on computation
    total = _sum(computations)

    native.genrule(
        name = name,
        outs = [name + ".out"],
        cmd = "echo 'Generated with computation: {}' > $@".format(total),
        **kwargs
    )

def generate_many_targets(prefix, count):
    """Generate many targets to increase Starlark load."""
    for i in range(count):
        target_name = "{}_{}".format(prefix, i)

        # Heavy computation per target
        h = _compute_hash(target_name)
        matrix_val = _matrix_computation(20)

        native.genrule(
            name = target_name,
            outs = [target_name + ".txt"],
            cmd = "echo 'Target {} computed: {}' > $@".format(i, h + matrix_val),
        )

def _string_processing(s, iterations):
    """Heavy string processing."""
    result = s
    for _ in range(iterations):
        result = result.replace("a", "b").replace("b", "c").replace("c", "a")
        result = result.upper().lower()
        h = _compute_hash(result)
        result = result + str(h % 100)
    return result[:100]  # Truncate

def process_and_generate(name, base_string):
    """Process strings heavily and generate a target."""
    processed = _string_processing(base_string * 10, 50)

    native.genrule(
        name = name,
        outs = [name + ".processed"],
        cmd = "echo '{}' > $@".format(processed),
    )
