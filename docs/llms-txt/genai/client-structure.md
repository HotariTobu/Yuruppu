# Client Structure

The `Client` provides access to various services:

```go
type Client struct {
    Models           *Models            // Content generation
    Live             *Live              // Real-time WebSocket
    Caches           *Caches            // Model response caching
    Chats            *Chats             // Chat session utilities
    Files            *Files             // File upload/management
    Operations       *Operations        // Long-running operations
    FileSearchStores *FileSearchStores  // RAG file stores
    Batches          *Batches           // Batch processing
    Tunings          *Tunings           // Model fine-tuning
    AuthTokens       *Tokens            // Authentication tokens
}
```
