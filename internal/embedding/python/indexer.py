#!/usr/bin/env python3
import argparse
import json
import sys
import uuid
import time
from typing import Dict, List, Any

import chromadb
from sentence_transformers import SentenceTransformer


def process_request(client: chromadb.HttpClient, req: str, model: SentenceTransformer) -> Dict[str, Any]:
    req_id = str(uuid.uuid4())
    try:
        input_data = json.loads(req)
        req_id = input_data.get("meta", {}).get("id", req_id)
        chunks = input_data.get("chunks", [])

        if chunks:
            result = index_chunks(client, req_id, chunks, model)
        else:
            result = {"id": req_id, "status": "error", "message": "No chunks provided"}

    except json.JSONDecodeError as e:
        result = {"id": req_id, "status": "error", "message": f"Invalid JSON: {str(e)}"}
    except Exception as e:
        result = {"id": req_id, "status": "error", "message": str(e)}

    return result


def index_chunks(client: chromadb.HttpClient, req_id: str, chunks: List[Dict[str, str]], model: SentenceTransformer):
    # Get or create collection (thread-safe with server mode)
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

    # Upsert is thread-safe in server mode
    collection.upsert(
        ids=ids,
        embeddings=embeddings.tolist(),
        documents=documents,
        metadatas=metadata_list,
    )

    return {"id": req_id, "status": "success", "indexed_count": len(chunks)}


def wait_for_server(host: str, port: int, timeout: int = 30):
    start_time = time.time()
    while time.time() - start_time < timeout:
        # noinspection PyBroadException
        try:
            client = chromadb.HttpClient(host=host, port=port)
            client.heartbeat()  # Test connection
            print(f"✓ ChromaDB server is available at {host}:{port}", file=sys.stderr)
            return True
        except Exception:
            time.sleep(0.2)

    print(f"✗ ChromaDB server not available at {host}:{port} after {timeout}s", file=sys.stderr)
    return False


def main():
    parser = argparse.ArgumentParser(description="Index code chunks in ChromaDB (Server Mode)")
    parser.add_argument(
        "--host",
        default="localhost",
        help="ChromaDB server host (default: localhost)"
    )
    parser.add_argument(
        "--port",
        type=int,
        default=8000,
        help="ChromaDB server port (default: 8000)"
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=30,
        help="Server connection timeout in seconds (default: 30)"
    )
    parser.add_argument(
        "--model-name",
        default="all-MiniLM-L6-v2",
        help="Name of the sentence transformer model (default: all-MiniLM-L6-v2)"
    )
    args = parser.parse_args()

    if not wait_for_server(args.host, args.port, args.timeout):
        print("Unable to join chroma server, is it started?", file=sys.stderr)
        sys.exit(1)

    try:
        model = SentenceTransformer(args.model_name, local_files_only=True)
        print(f"✓ Loaded model '{args.model_name}' from cache", file=sys.stderr)
    except Exception as e:
        print(f"✗ Failed to load model '{args.model_name}' from cache: {e}", file=sys.stderr)
        print("Please run: python cache_model.py <model_name> first", file=sys.stderr)
        sys.exit(1)

    try:
        client = chromadb.HttpClient(host=args.host, port=args.port)
        print(f"✓ Connected to ChromaDB server at {args.host}:{args.port}", file=sys.stderr)
    except Exception as e:
        print(f"✗ Failed to connect to ChromaDB server: {e}", file=sys.stderr)
        sys.exit(1)

    print(json.dumps({"status": "READY"}))
    sys.stdout.flush()

    while True:
        line = sys.stdin.readline()
        if not line:
            break

        request = line.strip()
        if not request or request == "exit":
            break

        result = process_request(client, request, model)

        print(json.dumps(result))
        sys.stdout.flush()


if __name__ == "__main__":
    main()