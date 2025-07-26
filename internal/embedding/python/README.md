## Cache model

The cache_model.py script is designed to help you cache a Hugging Face model for offline use, 
ensuring that the python indexer can run without making network requests.

### Usage:

1. **First, cache your model:**
```bash
python cache_model.py all-MiniLM-L6-v2
```

2. **Check if a model is cached:**
```bash
python cache_model.py --check all-MiniLM-L6-v2
```

3. **List common models:**
```bash
python cache_model.py --list
```

4. **Cache a different model:**
```bash
python cache_model.py all-mpnet-base-v2
```

5. **Force re-download:**
```bash
python cache_model.py all-MiniLM-L6-v2 --force
```

6. **Run your indexer with cached model:**
```bash
python indexer.py --model-name all-MiniLM-L6-v2
```

The caching script will:
- Download and cache the model
- Verify it can be loaded offline
- Test that encoding works
- Show you where the cache is stored


## Chroma manager

Script to manage the ChromaDB server, including starting, stopping, and checking status.

### Usage

**1. Start ChromaDB Server:**
```bash
python chroma_manager.py start --path ~/.mm/chroma
```

**2. Check Server Status:**
```bash
python chroma_manager.py status
```

**3. Stop Server When Done:**
```bash
python chroma_manager.py stop
```
