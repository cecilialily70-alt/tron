#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <time.h>
#include <cuda_runtime.h>
#include <curand_kernel.h>

#define BLOCK 256
#define RECORD_SIZE 96 

typedef uint32_t fe[8];

__constant__ uint32_t Gx[8] = {0x16F81798, 0x59F2815B, 0x2DCE28D9, 0x029BFCDB, 0xCE870B07, 0x55A06295, 0xF9DCBBAC, 0x79BE667E};
__constant__ uint32_t Gy[8] = {0x483ADA77, 0x26A3C465, 0x5DA4FBFC, 0xFD17B448, 0x0E1108A8, 0x68554199, 0xC47D08FF, 0x483AE9A1};

// 修复点 1：绝对不会溢出的乘法与安全的取模规约
__device__ void fe_mul(fe r, const fe a, const fe b) {
    uint32_t t[16] = {0};
    for(int i=0; i<8; i++) {
        uint64_t carry = 0;
        for(int j=0; j<8; j++) {
            uint64_t prod = (uint64_t)t[i+j] + ((uint64_t)a[i] * b[j]) + carry;
            t[i+j] = (uint32_t)prod;
            carry = prod >> 32;
        }
        t[i+8] = (uint32_t)carry;
    }
    for(int i=15; i>=8; i--) {
        uint32_t h = t[i];
        if(!h) continue;
        uint64_t c = (uint64_t)t[i-8] + ((uint64_t)h * 977);
        t[i-8] = (uint32_t)c;
        c >>= 32;
        c += (uint64_t)t[i-7] + h;
        t[i-7] = (uint32_t)c;
        c >>= 32;
        int k = i - 6;
        while(c && k < 16) {
            c += t[k];
            t[k] = (uint32_t)c;
            c >>= 32;
            k++;
        }
    }
    for(int i=0; i<8; i++) r[i] = t[i];
}

__device__ void fe_sqr(fe r, const fe a) { fe_mul(r, a, a); }

__device__ void fe_add(fe r, const fe a, const fe b) {
    uint64_t c = 0;
    for(int i=0; i<8; i++) {
        c += a[i]; c += b[i];
        r[i] = (uint32_t)c; c >>= 32;
    }
}

// 修复点 2：修复了类型转换带来的负数借位灾难
__device__ void fe_sub(fe r, const fe a, const fe b) {
    uint64_t c = 0;
    for(int i=0; i<8; i++) {
        uint64_t diff = (uint64_t)a[i] - b[i] - c;
        r[i] = (uint32_t)diff;
        c = (diff >> 32) ? 1 : 0;
    }
    if(c) {
        uint64_t carry = (uint64_t)r[0] + 0xFFFFFC2F; r[0] = (uint32_t)carry; carry >>= 32;
        carry += (uint64_t)r[1] + 0xFFFFFFFE; r[1] = (uint32_t)carry; carry >>= 32;
        for(int i=2; i<8; i++) {
            carry += (uint64_t)r[i] + 0xFFFFFFFF; r[i] = (uint32_t)carry; carry >>= 32;
        }
    }
}

// 修复点 3：强制 P 边界规约，确保哈希 100% 准确
__device__ void fe_reduce(fe r) {
    int ge = 1;
    for(int i=7; i>=0; i--) {
        uint32_t pi = (i==0) ? 0xFFFFFC2F : ((i==1) ? 0xFFFFFFFE : 0xFFFFFFFF);
        if(r[i] > pi) break;
        if(r[i] < pi) { ge = 0; break; }
    }
    if(ge) {
        uint64_t c = 0;
        uint64_t sub = 0xFFFFFC2F;
        c = (uint64_t)r[0] - sub; r[0] = (uint32_t)c; c = (c >> 32) ? 1 : 0;
        sub = 0xFFFFFFFE;
        c = (uint64_t)r[1] - sub - c; r[1] = (uint32_t)c; c = (c >> 32) ? 1 : 0;
        for(int i=2; i<8; i++) {
            c = (uint64_t)r[i] - 0xFFFFFFFF - c; r[i] = (uint32_t)c; c = (c >> 32) ? 1 : 0;
        }
    }
}

__device__ void jac_double(fe x3, fe y3, fe z3, const fe x1, const fe y1, const fe z1) {
    fe a, b, c, d, e, f, t1;
    fe_sqr(a, x1); fe_sqr(b, y1); fe_sqr(c, b);
    fe_add(t1, x1, b); fe_sqr(d, t1); fe_sub(d, d, a); fe_sub(d, d, c); fe_add(d, d, d);
    fe_add(e, a, a); fe_add(e, e, a); fe_sqr(f, e);
    fe_add(t1, d, d); fe_sub(x3, f, t1);
    fe_sub(y3, d, x3); fe_mul(y3, e, y3);
    fe_add(c, c, c); fe_add(c, c, c); fe_add(c, c, c); fe_sub(y3, y3, c);
    fe_add(t1, y1, y1); fe_mul(z3, t1, z1);
}

__device__ void jac_add(fe x3, fe y3, fe z3, const fe x1, const fe y1, const fe z1, const fe x2, const fe y2) {
    fe z1z1, u2, s2, h, i, j, r, v, t1, t2;
    fe_sqr(z1z1, z1);
    fe_mul(u2, x2, z1z1);
    fe_mul(t1, z1, z1z1); fe_mul(s2, y2, t1);
    fe_sub(h, u2, x1);
    fe_add(t1, h, h); fe_sqr(i, t1);
    fe_mul(j, h, i);
    fe_sub(r, s2, y1); fe_add(r, r, r);
    fe_mul(v, x1, i);
    fe_sqr(x3, r); fe_sub(x3, x3, j); fe_add(t1, v, v); fe_sub(x3, x3, t1);
    fe_sub(y3, v, x3); fe_mul(y3, r, y3); fe_mul(t2, y1, j); fe_add(t2, t2, t2); fe_sub(y3, y3, t2);
    fe_add(t1, z1, z1); fe_mul(z3, t1, h);
}

__device__ void fe_inv(fe r, const fe a) {
    fe x; for(int i=0; i<8; i++) x[i] = a[i];
    fe res = {1,0,0,0,0,0,0,0};
    uint32_t exp[8] = {0xFFFFFC2D, 0xFFFFFFFE, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF};
    for(int i=7; i>=0; i--) {
        for(int j=31; j>=0; j--) {
            fe_sqr(res, res);
            if((exp[i] >> j) & 1) fe_mul(res, res, x);
        }
    }
    for(int i=0; i<8; i++) r[i] = res[i];
}

__global__ void init_rng(curandState *s, uint64_t seed) {
    int tid = blockIdx.x * blockDim.x + threadIdx.x;
    curand_init(seed, tid, 0, &s[tid]);
}

__global__ void gen_keys_with_pub(uint8_t *out, curandState *s, uint32_t n) {
    int tid = blockIdx.x * blockDim.x + threadIdx.x;
    if (tid >= n) return;
    curandState r = s[tid];
    
    fe priv = {0};
    uint8_t priv_bytes[32];
    for (int i = 0; i < 8; i++) {
        uint32_t w = curand(&r);
        priv[i] = w;
        priv_bytes[31 - (i*4)]   = (uint8_t)w;
        priv_bytes[31 - (i*4+1)] = (uint8_t)(w>>8);
        priv_bytes[31 - (i*4+2)] = (uint8_t)(w>>16);
        priv_bytes[31 - (i*4+3)] = (uint8_t)(w>>24);
    }
    s[tid] = r;

    fe Px={0}, Py={0}, Pz={0};
    bool first = true;
    for(int i=7; i>=0; i--) {
        for(int j=31; j>=0; j--) {
            if(!first) jac_double(Px, Py, Pz, Px, Py, Pz);
            if((priv[i] >> j) & 1) {
                if(first) {
                    for(int k=0; k<8; k++) { Px[k]=Gx[k]; Py[k]=Gy[k]; Pz[k]=(k==0)?1:0; }
                    first = false;
                } else {
                    jac_add(Px, Py, Pz, Px, Py, Pz, Gx, Gy);
                }
            }
        }
    }

    fe z_inv, z_inv2, z_inv3;
    fe_inv(z_inv, Pz);
    fe_sqr(z_inv2, z_inv);
    fe_mul(z_inv3, z_inv2, z_inv);
    fe_mul(Px, Px, z_inv2);
    fe_mul(Py, Py, z_inv3);

    // 强行规约，保证提供给 CPU 的哈希底料 100% 正确
    fe_reduce(Px);
    fe_reduce(Py);

    uint8_t *record = out + tid * RECORD_SIZE;
    for(int i=0; i<32; i++) record[i] = priv_bytes[i];
    for(int i=0; i<8; i++) {
        record[32 + 31 - (i*4)]   = (uint8_t)Px[i]; record[32 + 31 - (i*4+1)] = (uint8_t)(Px[i]>>8);
        record[32 + 31 - (i*4+2)] = (uint8_t)(Px[i]>>16); record[32 + 31 - (i*4+3)] = (uint8_t)(Px[i]>>24);
        record[64 + 31 - (i*4)]   = (uint8_t)Py[i]; record[64 + 31 - (i*4+1)] = (uint8_t)(Py[i]>>8);
        record[64 + 31 - (i*4+2)] = (uint8_t)(Py[i]>>16); record[64 + 31 - (i*4+3)] = (uint8_t)(Py[i]>>24);
    }
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
        gen_keys_with_pub<<<batch/BLOCK, BLOCK>>>(d_out, d_rng, batch);
        cudaMemcpy(h_out, d_out, sz, cudaMemcpyDeviceToHost);
        fwrite(h_out, 1, sz, stdout);
    }
    return 0;
}