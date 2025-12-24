# ADR: Cloud Build Image for ko

> Date: 2025-12-24
> Status: **Adopted**

## Context

The current Cloud Build configuration uses the official golang image and installs ko on every build via `go install`. This adds unnecessary build time (~15-30 seconds) and network dependency. We need to select a pre-built image with ko pre-installed for faster, more reliable builds.

## Decision Drivers

- Eliminate ko installation overhead per build
- Maintain security and supply chain integrity
- Minimize external dependencies
- Compatible with Google Artifact Registry authentication

## Options Considered

- **Option 1:** ghcr.io/ko-build/ko (Official)
- **Option 2:** cgr.dev/chainguard/ko (Chainguard)
- **Option 3:** gcr.io/$PROJECT_ID/ko (Community Builder)
- **Option 4:** golang + ko install (Current baseline)

## Evaluation

Criteria adapted for build tooling (container images, not Go libraries):

| Criterion | Weight | ko-build/ko | Chainguard | Community | golang+install |
|-----------|--------|-------------|------------|-----------|------------------|
| Functional Fit | 25% | 5 (1.25) | 5 (1.25) | 4 (1.00) | 4 (1.00) |
| Go Compatibility | 20% | 5 (1.00) | 5 (1.00) | 3 (0.60) | 5 (1.00) |
| Lightweight | 15% | 4 (0.60) | 5 (0.75) | 3 (0.45) | 3 (0.45) |
| Security | 15% | 4 (0.60) | 5 (0.75) | 3 (0.45) | 4 (0.60) |
| Documentation | 15% | 4 (0.60) | 5 (0.75) | 3 (0.45) | 5 (0.75) |
| Ecosystem | 10% | 5 (0.50) | 4 (0.40) | 2 (0.20) | 5 (0.50) |
| **Total** | 100% | **4.55** | **4.90** | **3.15** | **4.30** |

### Notes

- **ko-build/ko**: Official image, well-maintained, but no supply chain attestations
- **Chainguard**: Security-focused with signed images, SBOM, minimal CVEs. Free tier available for developer use
- **Community Builder**: Requires manual build in your GCP project, uses outdated Go version
- **golang+install**: Simple but adds 15-30s overhead per build

## Decision

Adopt **cgr.dev/chainguard/ko**.

## Rationale

1. **Security-first**: Chainguard images are built with security as priority - signed with Sigstore, includes SBOM, minimal attack surface

2. **Minimal CVEs**: Chainguard actively patches vulnerabilities, often achieving zero known CVEs

3. **Pre-installed tooling**: Includes ko, Go, and build-base - everything needed for `ko build`

4. **Free tier available**: Developer tier is free for personal/development use

5. **Smaller image size**: Chainguard images are typically smaller than traditional images due to minimal base

## Consequences

**Positive:**
- Faster builds (no ko installation step)
- Enhanced security posture with signed images
- SBOM included for supply chain transparency
- Regular security updates from Chainguard

**Negative:**
- External dependency on Chainguard registry
- May require paid tier for production at scale
- Less familiar than official golang images

**Risks:**
- **Risk**: Chainguard registry availability
- **Mitigation**: Image can be mirrored to Artifact Registry if needed

## Related Decisions

- [20251218-cicd.md](./20251218-cicd.md) - CI/CD uses Cloud Build
- [20251218-container-build.md](./20251218-container-build.md) - Uses ko for image builds

## Resources

| Option | Documentation | Registry |
|--------|---------------|----------|
| ko-build/ko | [ko.build](https://ko.build/) | [ghcr.io/ko-build/ko](https://github.com/ko-build/ko/pkgs/container/ko) |
| Chainguard ko | [Chainguard Academy](https://edu.chainguard.dev/chainguard/chainguard-images/tooling/building-go-containers-with-ko/) | [cgr.dev/chainguard/ko](https://images.chainguard.dev/directory/image/ko/overview) |

## Sources

- [Chainguard ko Image Overview](https://images.chainguard.dev/directory/image/ko/overview)
- [Building Go Containers with ko - Chainguard Academy](https://edu.chainguard.dev/chainguard/chainguard-images/tooling/building-go-containers-with-ko/)
- [ko Official Documentation](https://ko.build/)
- [Ship Go apps faster to Cloud Run with ko - Google Cloud Blog](https://cloud.google.com/blog/topics/developers-practitioners/ship-your-go-applications-faster-cloud-run-ko)
