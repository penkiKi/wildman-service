# Third-party notices

Wildman Service depends directly on the following open-source packages. Transitive packages and exact versions are recorded by `go.sum`, `frontend/bun.lock`, and the generated SPDX SBOM.

| Component | License |
|---|---|
| github.com/go-chi/chi/v5 | MIT |
| golang.org/x/crypto | BSD-3-Clause |
| modernc.org/sqlite | BSD-3-Clause |
| React / React DOM | MIT |
| lucide-react | ISC |
| Tailwind CSS / @tailwindcss/vite | MIT |
| Vite / @vitejs/plugin-react | MIT |
| TypeScript and React type definitions | Apache-2.0 / MIT |

Build and runtime images use Oven Bun, Go, and Debian components under their respective upstream licenses. Distribution must include notices required by the exact dependency versions represented in the release SBOM. MusicBrainz data licensing and attribution are documented separately in `docs/PROVIDER_MUSICBRAINZ.md`.
