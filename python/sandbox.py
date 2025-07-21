import time

import chromadb

# Connect to server
client = chromadb.HttpClient(host='localhost', port=8000)

# Create a collection
try:
    client.delete_collection("test")
except:
    pass
collection = client.get_or_create_collection("test")

# Add some documents
collection.add(
    documents=["This is foobar document", "This is hello world document"],
    metadatas=[{"source": "test"}, {"source": "test"}],
    ids=["id1", "id2"]
)

query = "Where is foobar?"

# Query
# Get more results than you need
start = time.time()
results = collection.query(query_texts=[query], n_results=50)
end = time.time()
print("Query took {:.2f}ms".format((end - start) * 1000))


# Filter and take top matches
good_matches = [
    (doc, dist) for doc, dist in zip(results['documents'][0], results['distances'][0])
    if dist < 1.0  # Your threshold
][:5]  # Take top 5 after filtering

for i, (doc, dist) in enumerate(good_matches):
    print(f"Rank {i+1}: {doc}")
    print(f"  Similarity: {1 - results['distances'][0][i]:.3f}")  # Convert distance to similarity
    print(f"  ID: {results['ids'][0][i]}")
    print()
