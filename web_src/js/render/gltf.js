// import { ModelViewerElement } from '@google/model-viewer';

export async function initGltfViewer() {
  const els = document.querySelectorAll('.gltf-viewer');
  if (!els.length) return;

  await import(/* webpackChunkName: "@google/model-viewer" */'@google/model-viewer');
}
