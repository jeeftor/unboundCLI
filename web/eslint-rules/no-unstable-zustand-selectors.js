/**
 * Custom ESLint rule: no-unstable-zustand-selectors
 *
 * Flags `useStore()` selector functions that return new array/object references
 * on every call. Zustand v5 uses `Object.is` comparison by default — a selector
 * that returns `.filter()`, `.map()`, `.find()`, an object literal, or an array
 * literal will create a new reference each time → infinite re-render loop
 * (React error #185: Maximum update depth exceeded).
 *
 * Correct patterns:
 *   useStore((s) => s.count)              // primitive — stable
 *   useStore((s) => s.entries)             // direct reference — stable
 *   const entries = useStore((s) => s.entries);
 *   const filtered = useMemo(() => entries.filter(...), [entries]); // useMemo — stable
 *
 * Incorrect patterns (flagged by this rule):
 *   useStore((s) => s.entries.filter(...))  // new array every call
 *   useStore((s) => ({ a: s.a, b: s.b }))   // new object every call
 *   useStore((s) => [s.a, s.b])             // new array every call
 */

const ARRAY_METHODS = new Set([
  'filter', 'map', 'find', 'findIndex', 'findLast', 'findLastIndex',
  'reduce', 'reduceRight', 'slice', 'concat', 'sort', 'flat', 'flatMap',
  'toSorted', 'toReversed', 'toSpliced', 'with',
]);

/** @type {import('eslint').Rule.RuleModule} */
export default {
  meta: {
    type: 'problem',
    docs: {
      description: 'Disallow Zustand selectors that return new references (causes infinite re-render)',
      category: 'Possible Errors',
      recommended: true,
    },
    messages: {
      arrayMethod:
        'useStore selector returns a new array via .{{method}}() — this causes infinite re-renders. Use useMemo with raw state instead, or wrap with useShallow.',
      objectLiteral:
        'useStore selector returns a new object literal — this causes infinite re-renders. Use useMemo with raw state instead, or wrap with useShallow.',
      arrayLiteral:
        'useStore selector returns a new array literal — this causes infinite re-renders. Use useMemo with raw state instead, or wrap with useShallow.',
    },
    schema: [],
  },

  create(context) {
    // Track which variables are useMemo'd — those are safe.
    const memoizedVars = new Set();

    return {
      // Track useMemo calls: const x = useMemo(() => ..., [...])
      CallExpression(node) {
        if (
          node.callee.type === 'Identifier' &&
          node.callee.name === 'useMemo'
        ) {
          // Mark the variable being assigned, if any
          const parent = node.parent;
          if (
            parent &&
            parent.type === 'VariableDeclarator' &&
            parent.id.type === 'Identifier'
          ) {
            memoizedVars.add(parent.id.name);
          }
        }
      },

      // Check useStore() calls
      'CallExpression[callee.name="useStore"]'(node) {
        if (!node.arguments || node.arguments.length === 0) return;

        const selector = node.arguments[0];
        if (selector.type !== 'ArrowFunctionExpression' && selector.type !== 'FunctionExpression') return;

        const body = selector.body;

        // Check if the body is a direct CallExpression (e.g. s.entries.filter(...))
        if (body.type === 'CallExpression') {
          checkExpression(body, context);
          return;
        }

        // Check if the body is an object/array literal
        if (body.type === 'ObjectExpression') {
          context.report({
            node: body,
            messageId: 'objectLiteral',
          });
          return;
        }

        if (body.type === 'ArrayExpression') {
          context.report({
            node: body,
            messageId: 'arrayLiteral',
          });
          return;
        }

        // Check if body is a conditional/ternary — check both branches
        if (body.type === 'ConditionalExpression') {
          checkExpression(body.consequent, context);
          checkExpression(body.alternate, context);
          return;
        }

        // Check if body is a logical expression (e.g. s.foo || [])
        if (body.type === 'LogicalExpression') {
          checkExpression(body.right, context);
          return;
        }
      },
    };
  },
};

function checkExpression(expr, context) {
  if (!expr) return;

  // Array method calls: s.entries.filter(...)
  if (
    expr.type === 'CallExpression' &&
    expr.callee.type === 'MemberExpression' &&
    expr.callee.property.type === 'Identifier' &&
    ARRAY_METHODS.has(expr.callee.property.name)
  ) {
    context.report({
      node: expr,
      messageId: 'arrayMethod',
      data: { method: expr.callee.property.name },
    });
    return;
  }

  // Object literal
  if (expr.type === 'ObjectExpression') {
    context.report({
      node: expr,
      messageId: 'objectLiteral',
    });
    return;
  }

  // Array literal
  if (expr.type === 'ArrayExpression') {
    context.report({
      node: expr,
      messageId: 'arrayLiteral',
    });
    return;
  }

  // Nested call (e.g. s.items.map(...).filter(...))
  if (expr.type === 'CallExpression' && expr.callee.type === 'MemberExpression') {
    checkExpression(expr.callee.object, context);
  }
}
