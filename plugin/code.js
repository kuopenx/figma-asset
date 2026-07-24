figma.showUI(__html__, {
  width: 360,
  height: 240,
  title: 'Figma Asset',
});

figma.ui.onmessage = async (message) => {
  const { id, payload } = message;

  if (message.action === 'figma.exportNodePng') {
    await handleExportPng(id, payload);
  } else if (message.action === 'figma.exportNodeSvg') {
    await handleExportSvg(id, payload);
  }
};

async function handleExportPng(id, payload) {
  const exports = [];
  const errors = [];
  const node = await figma.getNodeByIdAsync(payload.nodeId);

  if (!node || !('exportAsync' in node)) {
    errors.push({ nodeId: payload.nodeId, message: 'Node not found or not exportable' });
  } else {
    for (const scale of payload.scales) {
      try {
        const bytes = await node.exportAsync({
          format: 'PNG',
          constraint: { type: 'SCALE', value: scale },
          contentsOnly: payload.contentsOnly !== false,
        });
        exports.push({
          scale,
          format: 'png',
          bytes: Array.from(bytes),
        });
      } catch (error) {
        errors.push({
          nodeId: payload.nodeId,
          scale,
          message: error instanceof Error ? error.message : String(error),
        });
      }
    }
  }

  figma.ui.postMessage({
    id,
    ok: errors.length === 0 || exports.length > 0,
    result: { exports, nodeName: node ? node.name : '' },
    errors,
  });
}

async function handleExportSvg(id, payload) {
  const exports = [];
  const errors = [];
  const node = await figma.getNodeByIdAsync(payload.nodeId);

  if (!node || !('exportAsync' in node)) {
    errors.push({ nodeId: payload.nodeId, message: 'Node not found or not exportable' });
  } else {
    try {
      const bytes = await node.exportAsync({
        format: 'SVG',
        svgOutlineText: payload.outlineText !== false,
        svgIdAttribute: payload.includeIds === true,
        svgSimplifyStroke: payload.simplifyStroke !== false,
        contentsOnly: payload.contentsOnly !== false,
      });
      exports.push({
        scale: 0,
        format: 'svg',
        bytes: Array.from(bytes),
      });
    } catch (error) {
      errors.push({
        nodeId: payload.nodeId,
        message: error instanceof Error ? error.message : String(error),
      });
    }
  }

  figma.ui.postMessage({
    id,
    ok: errors.length === 0 || exports.length > 0,
    result: { exports, nodeName: node ? node.name : '' },
    errors,
  });
}