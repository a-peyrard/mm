#!/usr/bin/env python3
import argparse
import json
import sys
from typing import Dict, List, Any

import chromadb
from sentence_transformers import SentenceTransformer

def process_request(req: str, model: SentenceTransformer, db_path="../chroma") -> Dict[str, Any]:
    try:
        input_data = json.loads(req)
        chunks = input_data.get("chunks", [])

        if chunks:
            result = index_chunks(chunks, model, db_path)
        else:
            result = {"status": "error", "message": "No chunks provided"}

    except json.JSONDecodeError as e:
        result = {"status": "error", "message": f"Invalid JSON: {str(e)}"}
    except Exception as e:
        result = {"status": "error", "message": str(e)}

    return result


def index_chunks(chunks: List[Dict[str, str]], model: SentenceTransformer, db_path="../chroma"):
    client = chromadb.PersistentClient(path=db_path)

    collection = client.get_or_create_collection(
        name="code_chunks",
        metadata={"description": "Code chunks for semantic search"}
    )

    ids = []
    documents = []
    metadata_list = []
    for chunk in chunks:
        ids.append(chunk["id"])
        documents.append(chunk["content"])
        metadata_list.append(chunk.get("metadata", {}))

    embeddings = model.encode(documents)

    collection.add(
        ids=ids,
        embeddings=embeddings.tolist(),
        documents=documents,
        metadatas=metadata_list,
    )

    return {"status": "success", "indexed_count": len(chunks)}


def main():
    print("Starting indexer...")
    parser = argparse.ArgumentParser(description="Index code chunks in ChromaDB")
    parser.add_argument(
        "--db-path",
        default="./chroma_db",
        help="Path to ChromaDB database (default: ./chroma_db)"
    )
    args = parser.parse_args()

    model_name = "all-MiniLM-L6-v2"
    model = SentenceTransformer(model_name)

    while True:
        line = sys.stdin.readline()
        if not line:
            break

        request = line.strip()
        if not request or request == "exit":
            break

        result = process_request(request, model, db_path=args.db_path)

        print(json.dumps(result))
        sys.stdout.flush()


if __name__ == "__main__":
    main()