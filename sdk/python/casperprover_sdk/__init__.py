"""CasperProver Python SDK.

Provides:
- :class:`ProverClient` — synchronous REST client.
- ``cp`` CLI (installed via ``pip install casperprover-sdk``).

Homepage: https://github.com/anna-stolbovskaja/CasperProver
"""

from .client import ProverClient

__all__ = ["ProverClient"]
__version__ = "0.1.0"
