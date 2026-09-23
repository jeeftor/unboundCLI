export function diagnosticsEmptyState(totalIssues: number, totalEntries: number) {
  if (totalIssues > 0) {
    return {
      title: 'No issues match these filters',
      detail: `${totalIssues} issue${totalIssues === 1 ? '' : 's'} exist across ${totalEntries} entries. Adjust the filters to review them.`,
    };
  }
  return {
    title: 'All clear!',
    detail: `No issues found across ${totalEntries} entries.`,
  };
}
