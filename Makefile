# Makefile â€?TRON Vanity Generator v6 (GPU pre-screen + CPU verify)
CUDA_CC := $(shell nvidia-smi --query-gpu=compute_cap --format=csv,noheader 2>/dev/null | head -1 | tr -d '.[:space:]' | grep -oE '^[0-9]+')
ifneq ($(CUDA_CC),)
  CUDA_ARCH ?= sm_$(CUDA_CC)
else
  CUDA_ARCH ?= sm_120
endif
NVCC       ?= nvcc
NVCC_FLAGS  = -O3 -arch=$(CUDA_ARCH) --use_fast_math -Xcompiler -march=native
GO          = go
GO_FLAGS    = -ldflags="-s -w"

GPU_BIN = gpu/vanity_worker
GPU_SRC = gpu/vanity.cu
GO_BIN  = tron-vanity

.PHONY: all gpu gov clean test-gpu info run help

all: gpu gov

gpu:
	@echo "-> Compiling GPU pre-screener ($(CUDA_ARCH))..."
	$(NVCC) $(NVCC_FLAGS) -o $(GPU_BIN) $(GPU_SRC) -lcurand
	@echo "OK $(GPU_BIN)"

gov:
	$(GO) mod tidy
	@echo "-> Building Go verifier..."
	$(GO) build $(GO_FLAGS) -o $(GO_BIN) main.go
	@echo "OK $(GO_BIN)"

run: all
	./$(GO_BIN)

test-gpu: gpu
	@echo "-> Testing GPU (5s)..."
	@timeout 5 ./$(GPU_BIN) --batch 262144 || true
	@echo "OK"

info:
	@echo "nvcc: $$(which nvcc 2>/dev/null || echo NOT FOUND)"
	@echo "go:   $$(which go 2>/dev/null || echo NOT FOUND)"
	@echo "arch: $(CUDA_ARCH)"
	@nvidia-smi --query-gpu=name,memory.total,compute_cap --format=csv 2>/dev/null || true

clean:
	rm -f $(GO_BIN) $(GPU_BIN)
	@echo "OK"

help:
	@echo "TRON Vanity Generator v6"
	@echo "  make        Build all"
	@echo "  make run    Build and run"
