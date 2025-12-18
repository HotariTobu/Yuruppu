# ADR: Container Image Build Strategy

> Date: 2025-12-18
> Status: **Adopted**

## Context

The deployable-server spec (FR-005) requires creating infrastructure code to build container images for Cloud Run deployment. We need to decide how to build container images for the Go LINE bot.

## Decision Drivers

- Smallest possible image size for fast Cloud Run cold starts
- Simple to maintain with minimal configuration
- Security (minimal attack surface, no CVEs)
- Cloud Run compatibility
- Alignment with project preference for standard/simple approaches

## Options Considered

- **Option 1:** ko (No Dockerfile)
- **Option 2:** Multi-Stage with Distroless
- **Option 3:** Multi-Stage with Alpine
- **Option 4:** Multi-Stage with Scratch

## Evaluation

Criteria adapted for infrastructure tooling (not Go libraries):

| Criterion | Weight | ko | Distroless | Alpine | Scratch |
|-----------|--------|-----|------------|--------|---------|
| Functional Fit | 25% | 5 (1.25) | 5 (1.25) | 5 (1.25) | 4 (1.00) |
| Go Compatibility | 20% | 5 (1.00) | 5 (1.00) | 4 (0.80) | 5 (1.00) |
| Lightweight | 15% | 5 (0.75) | 5 (0.75) | 5 (0.75) | 5 (0.75) |
| Security | 15% | 5 (0.75) | 5 (0.75) | 5 (0.75) | 5 (0.75) |
| Documentation | 15% | 4 (0.60) | 4 (0.60) | 4 (0.60) | 4 (0.60) |
| Ecosystem | 10% | 4 (0.40) | 5 (0.50) | 5 (0.50) | 5 (0.50) |
| **Total** | 100% | **4.75** | **4.85** | **4.65** | **4.60** |

### Image Size Comparison

| Option | Base Size | Final Size (Go app) |
|--------|-----------|---------------------|
| ko | ~2 MB (distroless) | 3-5 MB |
| Distroless | ~2 MB | 10-12 MB |
| Alpine | ~4 MB | 8-10 MB |
| Scratch | 0 MB | 7-10 MB |

## Decision

Adopt **ko (No Dockerfile)**.

## Rationale

1. **Smallest images**: ko produces 3-5 MB images (smallest of all options) by using distroless base and optimized Go binary builds

2. **Zero configuration**: No Dockerfile to maintain. Just `ko build ./cmd/server`

3. **Fastest builds**: ko bypasses Docker entirely, building images directly from Go source

4. **Cloud Run native**: Google recommends ko for Go apps on Cloud Run ([blog post](https://cloud.google.com/blog/topics/developers-practitioners/ship-your-go-applications-faster-cloud-run-ko))

5. **Supply chain security**: Automatic SBOM generation and SLSA support

6. **Alignment with project philosophy**: Simplest approach with least configuration

While Distroless scored slightly higher (4.85 vs 4.75), ko eliminates Dockerfile maintenance entirely and produces smaller images. The tool is mature (8.3k GitHub stars, CNCF Sandbox project) and actively maintained.

## Consequences

**Positive:**
- No Dockerfile to maintain
- Smallest possible images (3-5 MB)
- Fastest build times
- Automatic CA certs and timezone data (via distroless base)
- Works seamlessly with Cloud Build

**Negative:**
- Team must learn ko tool (minimal learning curve: ~5 minutes)
- No shell in resulting images (standard for distroless)
- Go-specific tool (not usable if project adds non-Go components)

**Risks:**
- **Risk**: ko is less familiar than Docker
- **Mitigation**: Documentation in README, ko is simpler than multi-stage Dockerfiles

## Confirmation

Success criteria:
- Container images are < 10 MB
- `ko build` completes without errors
- Cloud Run deployment succeeds

## Related Decisions

- [20251218-cicd.md](./20251218-cicd.md) - CI/CD uses ko for image builds

## Resources

| Option | Documentation | Repository |
|--------|---------------|------------|
| ko | [ko.build](https://ko.build/) | [ko-build/ko](https://github.com/ko-build/ko) |
| Distroless | [Distroless README](https://github.com/GoogleContainerTools/distroless) | [GoogleContainerTools/distroless](https://github.com/GoogleContainerTools/distroless) |
| Alpine | [Docker Hub](https://hub.docker.com/_/alpine) | [alpinelinux/docker-alpine](https://github.com/alpinelinux/docker-alpine) |

## Sources

- [Ship your Go applications faster to Cloud Run with ko - Google Cloud Blog](https://cloud.google.com/blog/topics/developers-practitioners/ship-your-go-applications-faster-cloud-run-ko)
- [ko: Easy Go Containers](https://ko.build/)
- [Container images simplified with ko - Snyk](https://snyk.io/blog/container-images-simplified-with-google-ko/)
- [Alpine, Distroless or Scratch - Google Cloud Medium](https://medium.com/google-cloud/alpine-distroless-or-scratch-caac35250e0b)
