import json
import subprocess
import tempfile
from typing import Dict, List

import chromadb
import pytest
from chromadb import QueryResult
from sentence_transformers import SentenceTransformer

from indexer import index_chunks


@pytest.fixture
def temp_path():
    with tempfile.TemporaryDirectory() as temp_dir:
        yield temp_dir

@pytest.fixture(scope="session")
def model() -> SentenceTransformer:
    return SentenceTransformer("all-MiniLM-L6-v2")


def run_indexer(input_data: List[Dict[str, str]], db_path: str):
    json_input = json.dumps({
        "chunks": input_data,
    })

    result = subprocess.run(
        ["uv", "run", "indexer.py", "--db-path=" + db_path],
        input=json_input,
        text=True,
        capture_output=True
    )

    if result.returncode != 0:
        pytest.fail(f"Indexer failed: {result.stderr}")

    print(f'raw output: {result.stdout}')
    return json.loads(result.stdout)

def search(query: str, db_path: str, n_results: int = 5) -> QueryResult:
    client = chromadb.PersistentClient(path=db_path)
    collection = client.get_collection("code_chunks")

    results = collection.query(
        query_texts=[query],
        n_results=n_results
    )

    return results


def describe_e2e_tests_for_index_chunks():
    def test_should_allow_to_index_and_search(temp_path):
        # GIVEN
        chunks = [
            {
                "id": "auth.js_func_validateToken_15",
                "type": "function",
                "content": "function validateToken(token) { return jwt.verify(token, secret); }",
                "metadata": {
                    "file_path": "/src/auth.js",
                    "function_name": "validateToken",
                    "language": "javascript"
                }
            },
            {
                "id": "utils.py_func_calculateTax_8",
                "type": "function",
                "content": "def calculate_tax(income): return income * 0.3 if income > 50000 else income * 0.2",
                "metadata": {
                    "file_path": "/src/utils.py",
                    "function_name": "calculate_tax",
                    "language": "python"
                }
            }
        ]

        # WHEN
        result = run_indexer(chunks, db_path=temp_path)
        assert result["status"] == "success"

        # THEN
        results = search(query="authentication token validation", db_path=temp_path, n_results=1)
        assert len(results["ids"][0]) > 0
        assert "validateToken" in results["documents"][0][0]


def describe_index_chunks():
    def test_should_allow_to_index(temp_path, model):
        # GIVEN
        chunks = [
            {
                "id": "auth.js_func_validateToken_15",
                "type": "function",
                "content": "function validateToken(token) { return jwt.verify(token, secret); }",
                "metadata": {
                    "file_path": "/src/auth.js",
                    "function_name": "validateToken",
                    "language": "javascript"
                }
            }
        ]

        # WHEN
        result = index_chunks(chunks=chunks, model=model, db_path=temp_path)

        # THEN
        assert result["status"] == "success"
        assert result["indexed_count"] == 1

    def test_should_be_able_search_indexed_documents(temp_path, model):
        # GIVEN
        chunks = [
            {
                "id": "auth.js_func_validateToken_15",
                "type": "function",
                "content": "function validateToken(token) { return jwt.verify(token, secret); }",
                "metadata": {
                    "file_path": "/src/auth.js",
                    "function_name": "validateToken",
                    "language": "javascript"
                }
            },
            {
                "id": "utils.py_func_calculateTax_8",
                "type": "function",
                "content": "def calculate_tax(income): return income * 0.3 if income > 50000 else income * 0.2",
                "metadata": {
                    "file_path": "/src/utils.py",
                    "function_name": "calculate_tax",
                    "language": "python"
                }
            }
        ]

        # WHEN
        result = index_chunks(chunks=chunks, model=model, db_path=temp_path)

        # THEN
        assert result["status"] == "success"

        results = search(query="function to calculate taxes", db_path=temp_path, n_results=1)
        assert len(results["ids"][0]) > 0
        assert "calculate_tax" in results["documents"][0][0]
