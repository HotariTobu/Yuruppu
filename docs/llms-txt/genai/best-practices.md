# Best Practices

1. **Use environment variables** for configuration in production
2. **Enable streaming** for longer responses to improve UX
3. **Implement retry logic** with exponential backoff for transient errors
4. **Count tokens** before requests to avoid exceeding limits
5. **Use system instructions** to improve response quality and consistency
6. **Cache responses** with `client.Caches` for repeated queries
7. **Handle errors gracefully** with proper error type checking
8. **Set appropriate timeouts** in context for long-running operations
