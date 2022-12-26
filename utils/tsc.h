#include <stdint.h>

#if defined(__arm__) || defined(__aarch32__) || defined(__arm64__) || defined(__aarch64__) || defined(_M_ARM)

uint64_t get_tsc()
{
    return 0;
}

#else

// only SGX2 support

uint64_t get_tsc()
{
    uint64_t a, d;
    asm volatile("rdtsc" : "=a"(a), "=d"(d));
    return (d << 32) | a;
    return 0;
}
#endif
