import '@xyflow/react/dist/style.css';

import ELK from 'elkjs/lib/elk.bundled.js';
import { useEffect, useMemo, useState } from 'react';
import {
  Background,
  BackgroundVariant,
  Controls,
  ReactFlow,
  ReactFlowProvider,
  useNodesState,
  useEdgesState,
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
  'elk.layered.spacing.nodeNodeBetweenLayers': '60',
  'elk.spacing.nodeNode': '30',
  'elk.layered.nodePlacement.strategy': 'NETWORK_SIMPLEX',
};

async function layoutNodes(
  nodes: DiagramNode[],
  edges: DiagramEdge[]
): Promise<{ nodes: DiagramNode[]; edges: DiagramEdge[] }> {
  const graph = {
    id: 'root',
    layoutOptions: elkOptions,
    children: nodes.map((n) => ({
      id: n.id,
      width: 160,
      height: 44,
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

export function ReactFlowDiagram({ steps, height = 200 }: { steps: Step[]; height?: number }) {
  const { nodes: initialNodes, edges: initialEdges } = useMemo(() => stepsToNodesEdges(steps), [steps]);
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);
  const [layouted, setLayouted] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLayouted(false);
    layoutNodes(initialNodes, initialEdges).then(({ nodes: ln, edges: le }) => {
      if (cancelled) return;
      setNodes(ln);
      setEdges(le);
      setLayouted(true);
    });
    return () => { cancelled = true; };
  }, [initialNodes, initialEdges, setNodes, setEdges]);

  return (
    <div className="react-flow-container" style={{ height }}>
      <ReactFlowProvider>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          fitView
          fitViewOptions={{ padding: 0.15, maxZoom: 1.2 }}
          proOptions={{ hideAttribution: true }}
          nodesDraggable={false}
          nodesConnectable={false}
          elementsSelectable={false}
          panOnDrag
          zoomOnScroll
        >
          <Background variant={BackgroundVariant.Dots} gap={16} size={1} />
          <Controls showInteractive={false} />
        </ReactFlow>
      </ReactFlowProvider>
    </div>
  );
}
