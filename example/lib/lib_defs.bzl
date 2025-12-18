# Library-specific Starlark macros with heavy computation

def _sum(items):
    """Sum a list of numbers."""
    total = 0
    for item in items:
        total += item
    return total

def _char_value(c):
    """Get numeric value for a character."""
    chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.-/"
    idx = chars.find(c)
    return idx if idx >= 0 else 0

def _fibonacci(n):
    """Compute fibonacci iteratively."""
    if n <= 1:
        return n
    a, b = 0, 1
    for _ in range(n - 1):
        a, b = b, a + b
    return b

def _is_prime(n):
    """Check if number is prime."""
    if n < 2:
        return False
    # Check up to sqrt(n) using iteration
    i = 2
    for _ in range(n):
        if i * i > n:
            break
        if n % i == 0:
            return False
        i += 1
    return True

def _find_primes(limit):
    """Find all primes up to limit."""
    primes = []
    for n in range(2, limit):
        if _is_prime(n):
            primes.append(n)
            # Extra computation per prime found
            h = n
            for _ in range(50):
                h = (h * 31 + n) % 2147483647
    return primes

def _complex_hash(items):
    """Compute a complex hash over items."""
    h = 17
    for item in items:
        for c in str(item).elems():
            h = (h * 31 + _char_value(c)) % 2147483647
        h = (h * _fibonacci(20)) % 2147483647
    return h

def cc_library_with_analysis(name, srcs = [], hdrs = [], deps = [], **kwargs):
    """A cc_library wrapper that does heavy Starlark analysis."""

    # Compute primes for "analysis"
    primes = _find_primes(500)

    # Analyze each source file
    analysis_results = []
    for src in srcs + hdrs:
        h = _complex_hash([src, name] + primes[:50])
        fib = _fibonacci(30)
        analysis_results.append({
            "file": src,
            "hash": h,
            "fib": fib,
            "prime_sum": _sum(primes[:20]),
        })

    # Compute aggregate metrics
    total_hash = _sum([r["hash"] for r in analysis_results])
    total_fib = _sum([r["fib"] for r in analysis_results])

    # Generate visibility based on "analysis"
    visibility = kwargs.pop("visibility", ["//visibility:public"])

    native.cc_library(
        name = name,
        srcs = srcs,
        hdrs = hdrs,
        deps = deps,
        visibility = visibility,
        **kwargs
    )

    # Also generate an analysis file
    native.genrule(
        name = name + "_analysis",
        outs = [name + "_analysis.txt"],
        cmd = "echo 'Analysis: hash={} fib={} files={}' > $@".format(
            total_hash,
            total_fib,
            len(analysis_results),
        ),
    )

def generate_library_variants(base_name, srcs, hdrs, count):
    """Generate multiple library variants with heavy computation."""

    primes = _find_primes(200)

    for i in range(count):
        variant_name = "{}_v{}".format(base_name, i)

        # Heavy computation per variant
        h = _complex_hash([variant_name, i] + primes)
        fib_sum = _sum([_fibonacci(j % 25) for j in range(50)])

        native.genrule(
            name = variant_name,
            outs = [variant_name + ".variant"],
            cmd = "echo '// variant {} hash={} fib={}' > $@".format(
                i, h, fib_sum,
            ),
        )
