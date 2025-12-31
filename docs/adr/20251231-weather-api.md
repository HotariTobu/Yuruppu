# ADR: Weather API Selection

> Date: 2025-12-31
> Status: **Adopted**

<!--
ADR records decisions only. Do NOT add:
- Configuration examples or code snippets
- Version numbers
- Setup instructions or commands
-->

## Context

The tool-calling-weather feature requires a weather API to retrieve forecast data for Japanese locations. The API will be called when users ask weather-related questions to the LINE bot.

## Decision Drivers

- Must be free (paid APIs are out of scope per spec)
- No API key required preferred (easy to try)
- Must support Japanese locations (Tokyo, Osaka, etc.)
- Commercial use allowed (Apache 2.0 or similar license)
- Simple integration with Go's standard library

## Options Considered

- **Option 1:** wttr.in
- **Option 2:** Open-Meteo (Standard/JMA)
- **Option 3:** OpenWeatherMap

## Evaluation

See `evaluation-criteria.md` for criteria definitions.

| Criterion | Weight | wttr.in | Open-Meteo | OpenWeatherMap |
|-----------|--------|---------|------------|----------------|
| Functional Fit | 25% | 4 (1.00) | 5 (1.25) | 4 (1.00) |
| Go Compatibility | 20% | 3 (0.60) | 5 (1.00) | 4 (0.80) |
| Lightweight | 15% | 5 (0.75) | 5 (0.75) | 4 (0.60) |
| Security | 15% | 3 (0.45) | 4 (0.60) | 4 (0.60) |
| Documentation | 15% | 3 (0.45) | 5 (0.75) | 5 (0.75) |
| Ecosystem | 10% | 4 (0.40) | 4 (0.40) | 5 (0.50) |
| **Total** | 100% | **3.65** | **4.75** | **4.25** |

## Decision

Adopt **wttr.in**.

## Rationale

While Open-Meteo scores higher technically, wttr.in was selected due to:

1. **License compatibility**: Apache 2.0 allows commercial use without restrictions. Open-Meteo's free tier is non-commercial only.
2. **Zero friction**: No API key required.
3. **Direct city name support**: Accepts "Tokyo" or "東京" directly without geocoding step.
4. **Sufficient for MVP**: Reliability concerns are acceptable for initial development.

## Consequences

**Positive:**
- Immediate integration without registration
- Simple HTTP GET with city name in URL
- Rich JSON response with current conditions and 3-day forecast
- Japanese city names and characters supported

**Negative:**
- No SLA or guaranteed uptime (free community service)
- Past downtime incidents documented
- No official Go SDK (must use net/http directly)

**Risks:**
- Service downtime: Implement caching (5-10 min TTL) and graceful error handling
- Rate limiting: Monitor usage, consider self-hosting if needed

## Related Decisions

None.

## Resources

| Option | Documentation | Repository |
|--------|---------------|------------|
| wttr.in | [GitHub README](https://github.com/chubin/wttr.in) | [chubin/wttr.in](https://github.com/chubin/wttr.in) |
| Open-Meteo | [Docs](https://open-meteo.com/en/docs) | [open-meteo/open-meteo](https://github.com/open-meteo/open-meteo) |
| OpenWeatherMap | [API Docs](https://openweathermap.org/api) | N/A |

## Sources

- [wttr.in GitHub](https://github.com/chubin/wttr.in)
- [wttr.in JSON API Documentation](https://github.com/chubin/wttr.in/issues/147)
- [Open-Meteo Pricing](https://open-meteo.com/en/pricing)
