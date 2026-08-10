import '@xyflow/react/dist/style.css';

import ELK from 'elkjs/lib/elk.bundled.js';
import { useEffect, useMemo } from 'react';
import {
  Background,
  BackgroundVariant,
  Controls,
  EdgeLabelRenderer,
  getBezierPath,
  ReactFlow,
  ReactFlowProvider,
  useNodesState,
  useEdgesState,
  useReactFlow,
} from '@xyflow/react';
import {
  nodeTypes,
  stepsToNodesEdges,
  type DiagramEdge,
  type DiagramNode,
  type Step,
} from './ReactFlowDiagramNodes';

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
  const opts = {
    ...elkOptions,
    'elk.direction': 'DOWN',
    'elk.layered.spacing.nodeNodeBetweenLayers': '60',
    'elk.spacing.nodeNode': '40',
  };

  const graph = {
    id: 'root',
    layoutOptions: opts,
    children: nodes.map((n) => ({
      id: n.id,
      width: 200,
      height: 48,
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

// Custom edge component that renders an explicit arrow at the target end
function ArrowEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  label,
  _labelStyle,
  _labelBgStyle,
  _labelBgPadding,
  animated,
  data,
  _markerEnd,
}: any) {
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const isWarn = data?.arrowWarn;
  const color = isWarn ? '#ef596f' : '#9aa5b3';

  // Calculate arrow angle at target end
  const dx = targetX - sourceX;
  const dy = targetY - sourceY;
  const angle = Math.atan2(dy, dx);
  const arrowLen = 10;
  const arrowWidth = 6;

  // Arrow tip at target point, slightly offset back along the path
  const tipX = targetX;
  const tipY = targetY;
  const baseX = tipX - arrowLen * Math.cos(angle);
  const baseY = tipY - arrowLen * Math.sin(angle);
  const leftX = baseX - arrowWidth * Math.sin(angle);
  const leftY = baseY + arrowWidth * Math.cos(angle);
  const rightX = baseX + arrowWidth * Math.sin(angle);
  const rightY = baseY - arrowWidth * Math.cos(angle);

  return (
    <g className={`react-flow__edge ${isWarn ? 'rf-edge-warn' : ''}`}>
      <path
        id={id}
        className={`react-flow__edge-path ${animated ? 'animated' : ''}`}
        d={edgePath}
        style={{ stroke: color, strokeWidth: 2, fill: 'none' }}
      />
      {/* Explicit arrow triangle at target end */}
      <path
        d={`M ${tipX} ${tipY} L ${leftX} ${leftY} L ${rightX} ${rightY} Z`}
        style={{ fill: color, stroke: color, strokeWidth: 1 }}
      />
      {label && (
        <EdgeLabelRenderer>
          <div
            style={{
              position: 'absolute',
              transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`,
              background: 'var(--bg-1)',
              border: '1px solid var(--border-subtle)',
              borderRadius: '3px',
              padding: '2px 6px',
              fontSize: '11px',
              fontWeight: 600,
              color: isWarn ? '#ef596f' : 'var(--text-2)',
              pointerEvents: 'none',
              whiteSpace: 'nowrap',
            }}
            className="rf-edge-label"
          >
            {label}
          </div>
        </EdgeLabelRenderer>
      )}
    </g>
  );
}

const edgeTypes = { arrow: ArrowEdge };

function FlowDiagramInner({ steps, height }: { steps: Step[]; height: number }) {
  // Always use vertical layout — reads top-to-bottom alongside the step list
  const useVertical = true;
  const { nodes: initialNodes, edges: initialEdges } = useMemo(
    () => stepsToNodesEdges(steps, useVertical),
    [steps, useVertical]
  );
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);
  const { fitView } = useReactFlow();

  // Set edge type to our custom arrow edge
  useEffect(() => {
    setEdges(eds => eds.map(e => ({ ...e, type: 'arrow' })));
  }, [setEdges, initialEdges]);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    void layoutNodes(initialNodes, initialEdges).then(({ nodes: ln, edges: le }) => {
      if (cancelled) return;
      setNodes(ln);
      setEdges(le.map(e => ({ ...e, type: 'arrow' })));
      timer = setTimeout(() => {
        if (cancelled) return;
        void fitView({ padding: 0.1, maxZoom: 1.5 });
      }, 50);
    });
    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
    };
  }, [initialNodes, initialEdges, setNodes, setEdges, fitView]);

  return (
    <div className="react-flow-container" style={{ height }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
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
