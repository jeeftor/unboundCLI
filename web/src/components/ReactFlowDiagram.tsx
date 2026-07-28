import '@xyflow/react/dist/style.css';

import ELK from 'elkjs/lib/elk.bundled.js';
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  Background,
  BackgroundVariant,
  Controls,
  ReactFlow,
  ReactFlowProvider,
  useNodesState,
  useEdgesState,
  useReactFlow,
} from '@xyflow/react';
import {
  nodeTypes,
  stepsToNodesEdges,
} from './ReactFlowDiagramNodes';
import type { DiagramEdge, DiagramNode, Step } from './ReactFlowDiagramNodes';

const elk = new ELK();

const elkOptions = {
  'elk.algorithm': 'layered',
  'elk.direction': 'RIGHT',
  'elk.layered.spacing.nodeNodeBetweenLayers': '40',
  'elk.spacing.nodeNode': '20',
  'elk.layered.nodePlacement.strategy': 'NETWORK_SIMPLEX',
};

async function layoutNodes(
  nodes: DiagramNode[],
  edges: DiagramEdge[]
): Promise<{ nodes: DiagramNode[]; edges: DiagramEdge[] }> {
  // Use vertical layout when there are many nodes (better fit for narrow modals)
  const useVertical = nodes.length > 4;
  const opts = useVertical
    ? { ...elkOptions, 'elk.direction': 'DOWN', 'elk.layered.spacing.nodeNodeBetweenLayers': '25' }
    : elkOptions;

  const graph = {
    id: 'root',
    layoutOptions: opts,
    children: nodes.map((n) => ({
      id: n.id,
      width: 200,
      height: 40,
    })),
    edges: edges.map((e) => ({
      id: e.id,
      sources: [e.source],
      targets: [e.target],
    })),
  };

  const layouted = await elk.layout(graph);

  const nodeMap = new Map((layouted.children ?? []).map((n) => [n.id, n]));
  const layoutedNodes = nodes.map((n) => {
    const pos = nodeMap.get(n.id);
    return {
      ...n,
      position: { x: pos?.x ?? 0, y: pos?.y ?? 0 },
    };
  });

  return { nodes: layoutedNodes, edges };
}

function FlowDiagramInner({ steps, height }: { steps: Step[]; height: number }) {
  const useVertical = steps.filter(s => s.nodeType).length > 4;
  const { nodes: initialNodes, edges: initialEdges } = useMemo(
    () => stepsToNodesEdges(steps, useVertical),
    [steps, useVertical]
  );
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);
  const { fitView } = useReactFlow();
  const layoutDone = useRef(false);

  useEffect(() => {
    let cancelled = false;
    layoutNodes(initialNodes, initialEdges).then(({ nodes: ln, edges: le }) => {
      if (cancelled) return;
      setNodes(ln);
      setEdges(le);
      // Fit view after layout completes — use setTimeout to ensure DOM updates first
      setTimeout(() => {
        fitView({ padding: 0.1, maxZoom: 1.5 });
      }, 50);
    });
    return () => { cancelled = true; };
  }, [initialNodes, initialEdges, setNodes, setEdges, fitView]);

  return (
    <div className="react-flow-container" style={{ height }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        proOptions={{ hideAttribution: true }}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={false}
        panOnDrag
        zoomOnScroll
        minZoom={0.1}
        maxZoom={2}
      >
        <Background variant={BackgroundVariant.Dots} gap={16} size={1} />
        <Controls showInteractive={false} />
      </ReactFlow>
    </div>
  );
}

export function ReactFlowDiagram({ steps, height = 180 }: { steps: Step[]; height?: number }) {
  return (
    <ReactFlowProvider>
      <FlowDiagramInner steps={steps} height={height} />
    </ReactFlowProvider>
  );
}
