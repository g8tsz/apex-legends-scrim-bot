// Angular CLI dev-server proxy configuration.
//
// The HuggingFace org hosting the scrim + league datasets is configurable so
// the app can be pointed at any HF account that mirrors the expected dataset
// layout. Set one of these env vars before running `npm start` to override:
//
//   HF_ORG              - HF org/user that owns both datasets
//                         (defaults to "Nexus-apex")
//   HF_SCRIMS_DATASET   - scrims dataset name (default "apex-scrims")
//   HF_LEAGUE_DATASET   - league dataset name (default "apex-league")
//   BACKEND_URL         - local Node backend for /leaderboard + /league
//                         (default http://localhost:3001)
//
// Example:
//   HF_ORG=my-org npm start
//
// Expected dataset layout:
//   https://huggingface.co/datasets/<HF_ORG>/<HF_SCRIMS_DATASET>
//     └── scrim_YYYY_MM_DD_id_*.json
//   https://huggingface.co/datasets/<HF_ORG>/<HF_LEAGUE_DATASET>
//     └── Season_N/Division_N/Week_*.json

const HF_ORG = process.env.HF_ORG || 'Nexus-apex';
const HF_SCRIMS_DATASET = process.env.HF_SCRIMS_DATASET || 'apex-scrims';
const HF_LEAGUE_DATASET = process.env.HF_LEAGUE_DATASET || 'apex-league';
const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:3001';

export default {
  '/leaderboard': {
    target: BACKEND_URL,
    secure: false,
  },
  '/league': {
    target: BACKEND_URL,
    secure: false,
  },
  '/hf-api': {
    target: 'https://huggingface.co',
    secure: true,
    changeOrigin: true,
    pathRewrite: { '^/hf-api': `/api/datasets/${HF_ORG}/${HF_SCRIMS_DATASET}` },
  },
  '/hf-resolve': {
    target: 'https://huggingface.co',
    secure: true,
    changeOrigin: true,
    pathRewrite: { '^/hf-resolve': `/datasets/${HF_ORG}/${HF_SCRIMS_DATASET}/raw/main` },
  },
  '/hf-league-api': {
    target: 'https://huggingface.co',
    secure: true,
    changeOrigin: true,
    pathRewrite: { '^/hf-league-api': `/api/datasets/${HF_ORG}/${HF_LEAGUE_DATASET}` },
  },
  '/hf-league-resolve': {
    target: 'https://huggingface.co',
    secure: true,
    changeOrigin: true,
    pathRewrite: { '^/hf-league-resolve': `/datasets/${HF_ORG}/${HF_LEAGUE_DATASET}/raw/main` },
  },
};
