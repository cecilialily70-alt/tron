// TRON Vanity GPU Worker v11
// Generates cryptographically random 32-byte private keys using cuRAND.
// All secp256k1, hashing, and address derivation is done by Go (trusted).
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <time.h>
#include <cuda_runtime.h>
#include <curand_kernel.h>

#define BLOCK 256
#define RECORD_SIZE 32 // only private key bytes

__global__ void init_rng(curandState *s, uint64_t seed) {
    int tid = blockIdx.x * blockDim.x + threadIdx.x;
    curand_init(seed, tid, 0, &s[tid]);
}

__global__ void gen_private_keys(uint8_t *out, curandState *s, uint32_t n) {
    int tid = blockIdx.x * blockDim.x + threadIdx.x;
    if (tid >= n) return;
    curandState r = s[tid];

    uint8_t *record = out + tid * RECORD_SIZE;
    for (int i = 0; i < 8; i++) {
        uint32_t w = curand(&r);
        record[31 - (i*4)]   = (uint8_t)w;
        record[31 - (i*4+1)] = (uint8_t)(w>>8);
        record[31 - (i*4+2)] = (uint8_t)(w>>16);
        record[31 - (i*4+3)] = (uint8_t)(w>>24);
    }
    s[tid] = r;
}

int main(int argc, char **argv) {
    uint32_t batch = 1048576;
    for (int i = 1; i < argc; i++)
        if (strcmp(argv[i], "--batch") == 0 && i+1 < argc) batch = atoi(argv[++i]);
    batch = ((batch + BLOCK - 1) / BLOCK) * BLOCK;

    cudaSetDevice(0);
    uint8_t *d_out, *h_out;
    size_t sz = (size_t)batch * RECORD_SIZE;
    cudaMalloc(&d_out, sz);
    h_out = (uint8_t*)malloc(sz);

    curandState *d_rng;
    cudaMalloc(&d_rng, batch * sizeof(curandState));
    init_rng<<<batch/BLOCK, BLOCK>>>(d_rng, time(NULL));
    cudaDeviceSynchronize();

    setvbuf(stdout, NULL, _IOFBF, 16*1024*1024);
    for (;;) {
        gen_private_keys<<<batch/BLOCK, BLOCK>>>(d_out, d_rng, batch);
        cudaMemcpy(h_out, d_out, sz, cudaMemcpyDeviceToHost);
        fwrite(h_out, 1, sz, stdout);
    }
    return 0;
}
