#include <stdint.h>

// only SGX2 support
uint64_t get_tsc()
{
    uint64_t a, d;
    asm volatile("rdtsc" : "=a"(a), "=d"(d));
    return (d << 32) | a;
}
