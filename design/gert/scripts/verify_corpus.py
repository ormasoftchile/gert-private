import json
import pathlib
import sys

import jsonschema
import yaml


def main() -> int:
    design_dir = pathlib.Path(__file__).resolve().parents[1]
    root = design_dir / "conformance"
    schema = json.loads((root / "schema.json").read_text(encoding="utf-8"))
    validator = jsonschema.Draft202012Validator(schema)

    errs = 0
    count = 0
    for f in sorted(root.glob("tv-*.yaml")):
        try:
            docs = list(yaml.safe_load_all(f.read_text(encoding="utf-8")))
        except Exception as e:
            print(f"FAIL {f.name} <yaml>: {e}")
            errs += 1
            continue

        for doc_index, doc in enumerate(docs, 1):
            vectors = doc.get("vectors") if isinstance(doc, dict) else None
            if not isinstance(vectors, list):
                print(f"FAIL {f.name} <doc {doc_index}>: missing vectors array")
                errs += 1
                continue

            for idx, vector in enumerate(vectors, 1):
                vector_id = vector.get("id", f"<vector {idx}>") if isinstance(vector, dict) else f"<vector {idx}>"
                try:
                    validator.validate({"vectors": [vector]})
                    count += 1
                except Exception as e:
                    message = e.message if hasattr(e, "message") else str(e)
                    print(f"FAIL {f.name} {vector_id}: {message}")
                    errs += 1

    total = count + errs
    if errs:
        print(f"FAIL {count}/{total} vectors validate")
        return 1

    print(f"OK {count}/{total} vectors validate")
    return 0


if __name__ == "__main__":
    sys.exit(main())
