from __future__ import annotations

import hashlib
import json
import struct
from dataclasses import dataclass
from pathlib import Path
from typing import Any


@dataclass(frozen=True)
class VoxModel:
    version: int
    size: tuple[int, int, int]
    voxel_count: int
    palette: tuple[tuple[int, int, int, int], ...] | None


def _read_chunks(data: bytes, start: int, end: int) -> list[tuple[bytes, bytes]]:
    chunks = []
    offset = start
    while offset < end:
        if end - offset < 12:
            raise ValueError("truncated VOX chunk header")
        chunk_id, content_size, children_size = struct.unpack_from("<4sII", data, offset)
        content_start = offset + 12
        content_end = content_start + content_size
        chunk_end = content_end + children_size
        if chunk_end > end:
            raise ValueError(f"chunk {chunk_id.decode('ascii', errors='replace')} exceeds its parent")
        chunks.append((chunk_id, data[content_start:content_end]))
        offset = chunk_end
    if offset != end:
        raise ValueError("VOX chunk sizes do not align with the parent boundary")
    return chunks


def inspect_vox(path: Path) -> VoxModel:
    """Validate one MagicaVoxel model and the subset used by the BN renderer."""
    data = path.read_bytes()
    if len(data) < 20 or data[:4] != b"VOX ":
        raise ValueError("not a MagicaVoxel VOX file")

    version = struct.unpack_from("<I", data, 4)[0]
    root_id, root_content_size, root_children_size = struct.unpack_from("<4sII", data, 8)
    if root_id != b"MAIN" or root_content_size != 0:
        raise ValueError("VOX file must contain an empty MAIN root chunk")
    children_start = 20
    children_end = children_start + root_children_size
    if children_end != len(data):
        raise ValueError("MAIN child size does not match the file length")

    chunks = _read_chunks(data, children_start, children_end)
    sizes = [content for chunk_id, content in chunks if chunk_id == b"SIZE"]
    voxel_chunks = [content for chunk_id, content in chunks if chunk_id == b"XYZI"]
    palette_chunks = [content for chunk_id, content in chunks if chunk_id == b"RGBA"]
    if len(sizes) != 1 or len(voxel_chunks) != 1:
        raise ValueError("renderer-compatible VOX assets must contain exactly one SIZE and one XYZI chunk")
    if len(palette_chunks) > 1:
        raise ValueError("VOX file contains multiple RGBA palette chunks")
    if len(sizes[0]) != 12:
        raise ValueError("SIZE chunk must contain exactly three 32-bit dimensions")

    dimensions = struct.unpack("<III", sizes[0])
    if any(dimension < 1 or dimension > 256 for dimension in dimensions):
        raise ValueError("model dimensions must be between 1 and 256 voxels")

    voxel_data = voxel_chunks[0]
    if len(voxel_data) < 4:
        raise ValueError("truncated XYZI chunk")
    voxel_count = struct.unpack_from("<I", voxel_data)[0]
    if len(voxel_data) != 4 + voxel_count * 4:
        raise ValueError("XYZI voxel count does not match the chunk length")
    if voxel_count > dimensions[0] * dimensions[1] * dimensions[2]:
        raise ValueError("XYZI voxel count exceeds the SIZE volume")
    occupied = set()
    for offset in range(4, len(voxel_data), 4):
        x, y, z, color_index = voxel_data[offset : offset + 4]
        if x >= dimensions[0] or y >= dimensions[1] or z >= dimensions[2]:
            raise ValueError("XYZI coordinate falls outside the SIZE dimensions")
        if color_index == 0:
            raise ValueError("XYZI color index 0 is reserved for transparency")
        coordinate = (x, y, z)
        if coordinate in occupied:
            raise ValueError(f"duplicate voxel coordinate {coordinate}")
        occupied.add(coordinate)

    palette = None
    if palette_chunks:
        if len(palette_chunks[0]) != 1024:
            raise ValueError("RGBA chunk must contain 256 RGBA colors")
        palette = tuple(
            tuple(palette_chunks[0][offset : offset + 4])
            for offset in range(0, 1024, 4)
        )

    return VoxModel(
        version=version,
        size=dimensions,
        voxel_count=voxel_count,
        palette=palette,
    )


def write_vox_manifest(
    input_path: Path,
    output_path: Path,
    *,
    asset_kind: str,
    asset_id: str | None = None,
) -> dict[str, Any]:
    model = inspect_vox(input_path)
    manifest = {
        "format": "bn3d-vox-asset-v1",
        "source": input_path.name,
        "sha256": hashlib.sha256(input_path.read_bytes()).hexdigest(),
        "asset_kind": asset_kind,
        "asset_id": asset_id,
        "vox_version": model.version,
        "dimensions": {"x": model.size[0], "y": model.size[1], "z": model.size[2]},
        "voxel_count": model.voxel_count,
        "has_embedded_palette": model.palette is not None,
        "renderer_compatibility": "single-model SIZE/XYZI subset",
    }
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    return manifest
