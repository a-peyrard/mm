#!/usr/bin/env python3
"""
Script to pre-download and cache sentence transformer models
Usage: python cache_model.py <model_name>
Example: python cache_model.py all-MiniLM-L6-v2
"""

import argparse
import sys
import os
from pathlib import Path
from sentence_transformers import SentenceTransformer


def calculate_model_size(model_name: str) -> float:
    """Calculate the total size of a cached model in MB"""
    try:
        # Common cache locations to check
        possible_cache_dirs = [
            os.path.expanduser("~/.cache/huggingface/hub"),
            os.path.expanduser("~/.cache/torch/sentence_transformers"),
            os.path.expanduser("~/.cache/sentence-transformers"),
        ]

        total_size = 0
        found = False

        for cache_dir in possible_cache_dirs:
            if not os.path.exists(cache_dir):
                continue

            # Look for directories containing the model name
            for item in os.listdir(cache_dir):
                if model_name.replace("/", "_") in item or model_name.replace("/", "--") in item:
                    model_path = os.path.join(cache_dir, item)
                    if os.path.isdir(model_path):
                        size = get_directory_size(model_path)
                        total_size += size
                        found = True
                        print(f"  Found model files in: {model_path} ({size / (1024*1024):.1f} MB)")

        if found:
            return total_size / (1024 * 1024)  # Convert to MB
        else:
            return None

    except Exception as e:
        print(f"  Could not calculate model size: {e}")
        return None


def get_directory_size(directory: str) -> int:
    """Get the total size of a directory in bytes"""
    total_size = 0
    try:
        for dirpath, dirnames, filenames in os.walk(directory):
            for filename in filenames:
                filepath = os.path.join(dirpath, filename)
                if os.path.exists(filepath):
                    total_size += os.path.getsize(filepath)
    except (OSError, IOError):
        pass
    return total_size


def cache_model(model_name: str, force_download: bool = False):
    """Download and cache a sentence transformer model"""

    print(f"Caching model: {model_name}")

    try:
        # Download the model (this will cache it locally)
        print("Downloading model...")
        if force_download:
            # Force re-download even if cached
            model = SentenceTransformer(model_name, cache_folder=None)
        else:
            model = SentenceTransformer(model_name)

        print(f"✓ Model '{model_name}' successfully cached")

        # Test that we can load it in offline mode
        print("Testing offline access...")
        offline_model = SentenceTransformer(model_name, local_files_only=True)
        print("✓ Model can be loaded in offline mode")

        # Show cache location info
        print("\nCache information:")
        hf_cache = os.path.expanduser("~/.cache/huggingface")
        if os.path.exists(hf_cache):
            print(f"HuggingFace cache directory: {hf_cache}")

        st_cache = os.path.expanduser("~/.cache/torch/sentence_transformers")
        if os.path.exists(st_cache):
            print(f"SentenceTransformers cache directory: {st_cache}")

        # Test encoding to make sure everything works
        print("\nTesting model functionality...")
        test_texts = ["Hello world", "This is a test"]
        embeddings = offline_model.encode(test_texts)
        print(f"✓ Model encoding works (output shape: {embeddings.shape})")

        # Calculate and display model size
        print("\nModel size information:")
        model_size_mb = calculate_model_size(model_name)
        if model_size_mb:
            print(f"✓ Model size: {model_size_mb:.1f} MB")

        return True

    except Exception as e:
        print(f"✗ Failed to cache model '{model_name}': {e}")
        return False


def list_common_models():
    """List some common sentence transformer models"""
    common_models = [
        ("all-MiniLM-L6-v2", "~90MB", "Fast and good quality"),
        ("all-mpnet-base-v2", "~420MB", "Best quality"),
        ("all-MiniLM-L12-v2", "~120MB", "Balanced speed/quality"),
        ("paraphrase-MiniLM-L6-v2", "~90MB", "Good for paraphrase detection"),
        ("distilbert-base-nli-mean-tokens", "~250MB", "Lightweight"),
        ("all-roberta-large-v1", "~1.3GB", "High quality, slower"),
    ]

    print("Common sentence transformer models:")
    print(f"{'Model Name':<35} {'Size':<10} {'Description'}")
    print("-" * 70)
    for model, size, desc in common_models:
        print(f"{model:<35} {size:<10} {desc}")


def check_model_cached(model_name: str):
    """Check if a model is already cached"""
    try:
        # Try to load in offline mode
        SentenceTransformer(model_name, local_files_only=True)
        print(f"✓ Model '{model_name}' is already cached")

        # Show size if cached
        model_size_mb = calculate_model_size(model_name)
        if model_size_mb:
            print(f"  Size: {model_size_mb:.1f} MB")

        return True
    except Exception:
        print(f"✗ Model '{model_name}' is not cached")
        return False


def main():
    parser = argparse.ArgumentParser(
        description="Download and cache sentence transformer models",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python cache_model.py all-MiniLM-L6-v2
  python cache_model.py all-mpnet-base-v2 --force
  python cache_model.py --list
  python cache_model.py --check all-MiniLM-L6-v2
        """
    )

    parser.add_argument(
        "model_name",
        nargs="?",
        help="Name of the sentence transformer model to cache"
    )

    parser.add_argument(
        "--force",
        action="store_true",
        help="Force re-download even if model is already cached"
    )

    parser.add_argument(
        "--list",
        action="store_true",
        help="List common sentence transformer models"
    )

    parser.add_argument(
        "--check",
        metavar="MODEL_NAME",
        help="Check if a specific model is already cached"
    )

    args = parser.parse_args()

    if args.list:
        list_common_models()
        return

    if args.check:
        check_model_cached(args.check)
        return

    if not args.model_name:
        parser.print_help()
        print("\nError: model_name is required unless using --list or --check")
        sys.exit(1)

    success = cache_model(args.model_name, args.force)
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()