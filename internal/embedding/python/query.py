import sys
import time

import chromadb

THRESHOLD = 1.3

if len(sys.argv) < 2:
    print("Usage: python query.py <query>")
    print("Example: python query.py 'where is foobar?'")
    sys.exit(1)

query = " ".join(sys.argv[1:])  # Join all arguments with spaces
print(f"Searching for: '{query}'")

client = chromadb.PersistentClient(path="/Users/augustin/.mm/chroma")
collection = client.get_or_create_collection("code_chunks")

total_docs = collection.count()
print(f"Total documents in collection: {total_docs}")

# Query
# Get more results than you need
start = time.time()
results = collection.query(query_texts=[query], n_results=50)
end = time.time()
print("Query took {:.2f}ms".format((end - start) * 1000))


# Filter and take top matches
good_matches = [
    (doc, dist) for doc, dist in zip(results['documents'][0], results['distances'][0])
    if dist < THRESHOLD  # Your threshold
][:5]  # Take top 5 after filtering

if not good_matches:
    print("No matches found within the threshold.")

for i, (doc, dist) in enumerate(good_matches):
    print(f"Rank {i+1}: {doc}")
    print(f"  Similarity: {THRESHOLD - results['distances'][0][i]:.3f}")  # Convert distance to similarity
    print(f"  ID: {results['ids'][0][i]}")
    print()
