<template>
  <div class="workflow-graph" v-if="jobs.length > 0">
    <div class="graph-header">
      <h4 class="graph-title">
        Workflow Dependencies
      </h4>
      <div class="graph-stats">
        {{ jobs.length }} jobs • {{ edges.length }} dependencies
        <span v-if="graphMetrics" class="graph-metrics">
          • {{ graphMetrics.successRate }} success • Parallelism: {{ graphMetrics.parallelism }}
        </span>
      </div>
      <div class="graph-controls">
        <button @click="resetView" class="control-btn" title="Reset view">
          <svg class="control-icon" viewBox="0 0 16 16" width="16" height="16">
            <path fill="currentColor" d="M8 3a5 5 0 1 0 4.546 2.914.5.5 0 0 1 .908-.417A6 6 0 1 1 8 2v1z"/>
          </svg>
        </button>
        <div class="zoom-controls">
          <button @click="zoomOut" class="control-btn" title="Zoom out">
            <svg class="control-icon" viewBox="0 0 16 16" width="16" height="16">
              <path fill="currentColor" d="M2.75 7.25a.75.75 0 0 0 0 1.5h10.5a.75.75 0 0 0 0-1.5H2.75z"/>
            </svg>
          </button>
          <span class="zoom-level">{{ Math.round(scale * 100) }}%</span>
          <button @click="zoomIn" class="control-btn" title="Zoom in">
            <svg class="control-icon" viewBox="0 0 16 16" width="16" height="16">
              <path fill="currentColor" d="M8.75 3.75a.75.75 0 0 0-1.5 0v3.5h-3.5a.75.75 0 0 0 0 1.5h3.5v3.5a.75.75 0 0 0 1.5 0v-3.5h3.5a.75.75 0 0 0 0-1.5h-3.5v-3.5z"/>
            </svg>
          </button>
        </div>
      </div>
    </div>

    <div
      class="graph-container"
      ref="container"
      @wheel="handleWheel"
      @mousedown="handleMouseDown"
      @mousemove="handleMouseMove"
      @mouseup="handleMouseUp"
      @mouseleave="handleMouseLeave"
      :class="{ 'dragging': isDragging }"
    >
      <svg
        :width="graphWidth"
        :height="graphHeight"
        class="graph-svg"
        :style="{
          transform: `translate(${translateX}px, ${translateY}px) scale(${scale})`,
          transformOrigin: '0 0'
        }"
      >
        <path
          v-for="edge in bezierEdges"
          :key="edge.key"
          :d="edge.path"
          fill="none"
          v-bind="getEdgeStyle(edge)"
          :class="[
            getEdgeClass(edge),
            { 'highlighted-edge': isEdgeHighlighted(edge) }
          ]"
        />

        <g
          v-for="job in jobsWithLayout"
          :key="job.id"
          class="job-node-group"
          :class="{
            'current-job': job.index === currentJobId
          }"
          @click="onNodeClick(job, $event)"
          @mouseenter="handleNodeMouseEnter(job)"
          @mouseleave="handleNodeMouseLeave"
        >
          <rect
            :x="job.x"
            :y="job.y"
            :width="nodeWidth"
            :height="nodeHeight"
            rx="8"
            :fill="getNodeColor(job.status)"
            :stroke="job.index === currentJobId ? 'var(--color-primary)' : 'var(--color-card-border)'"
            :stroke-width="job.index === currentJobId ? '3' : '2'"
            class="job-rect"
          />

          <rect
            v-if="job.status.toLowerCase() === 'running'"
            :x="job.x"
            :y="job.y"
            :width="nodeWidth"
            :height="nodeHeight"
            rx="8"
            fill="url(#running-gradient)"
            opacity="0.3"
            class="running-background"
          />

          <g
            :transform="`translate(${job.x + 12}, ${job.y + 12})`"
            class="status-icon"
          >
            <circle
              v-if="job.status.toLowerCase() === 'success'"
              r="6"
              :fill="getStatusDotColor('success')"
            />
            <polygon
              v-else-if="job.status.toLowerCase() === 'failure'"
              points="0,-5 5,3 -5,3"
              :fill="getStatusDotColor('failure')"
            />
            <circle
              v-else-if="job.status.toLowerCase() === 'running'"
              r="5"
              :fill="getStatusDotColor('running')"
              :class="{ 'pulse-dot': job.status.toLowerCase() === 'running' }"
            />
            <circle
              v-else
              r="4"
              :fill="getStatusDotColor('waiting')"
              :stroke="getStatusDotColor('waiting')"
              stroke-width="2"
            />
          </g>
          <text
            :x="job.x + 25"
            :y="job.y + 16"
            fill="white"
            font-size="12"
            text-anchor="start"
            class="job-name"
          >
            {{ getDisplayName(job.name) }}
          </text>

          <text
            v-if="job.duration && ['success', 'failure', 'completed'].includes(job.status.toLowerCase())"
            :x="job.x + nodeWidth - 10"
            :y="job.y + nodeHeight - 25"
            fill="rgba(255,255,255,0.7)"
            font-size="9"
            text-anchor="end"
            class="job-duration"
          >
            {{ job.duration }}
          </text>

          <text
            :x="job.x + nodeWidth - 10"
            :y="job.y + nodeHeight - 8"
            fill="rgba(255,255,255,0.9)"
            font-size="10"
            text-anchor="end"
            class="job-status"
          >
            {{ formatStatus(job.status) }}
          </text>

          <rect
            v-if="job.status.toLowerCase() === 'running'"
            :x="job.x + 2"
            :y="job.y + nodeHeight - 6"
            :width="(nodeWidth - 4) * 0.5"
            height="4"
            rx="2"
            :fill="getStatusDotColor('running')"
            class="progress-bar"
          >
            <animate
              attributeName="width"
              values="0; 100"
              dur="2s"
              repeatCount="indefinite"
              calcMode="spline"
              keySplines="0.4, 0, 0.2, 1"
            />
          </rect>

          <text
            v-if="job.needs && job.needs.length > 0"
            :x="job.x + nodeWidth / 2"
            :y="job.y - 8"
            fill="var(--color-text-light-2)"
            font-size="10"
            text-anchor="middle"
            class="job-deps-label"
          >
            ← {{ job.needs.length }} deps
          </text>
        </g>

        <defs>
          <linearGradient id="running-gradient" x1="0%" y1="0%" x2="100%" y2="0%">
            <stop offset="0%" :stop-color="getStatusDotColor('running')" stop-opacity="0.2"/>
            <stop offset="50%" :stop-color="getStatusDotColor('running')" stop-opacity="0.4"/>
            <stop offset="100%" :stop-color="getStatusDotColor('running')" stop-opacity="0.2"/>
          </linearGradient>

          <marker
            id="arrowhead-success"
            markerWidth="10"
            markerHeight="7"
            refX="9"
            refY="3.5"
            orient="auto"
          >
            <polygon points="0 0, 10 3.5, 0 7" :fill="getStatusDotColor('success')"/>
          </marker>

          <marker
            id="arrowhead-failure"
            markerWidth="10"
            markerHeight="7"
            refX="9"
            refY="3.5"
            orient="auto"
          >
            <polygon points="0 0, 10 3.5, 0 7" :fill="getStatusDotColor('failure')"/>
          </marker>

          <marker
            id="arrowhead-running"
            markerWidth="10"
            markerHeight="7"
            refX="9"
            refY="3.5"
            orient="auto"
          >
            <polygon points="0 0, 10 3.5, 0 7" :fill="getStatusDotColor('running')"/>
          </marker>

          <marker
            id="arrowhead-ready"
            markerWidth="10"
            markerHeight="7"
            refX="9"
            refY="3.5"
            orient="auto"
          >
            <polygon points="0 0, 10 3.5, 0 7" fill="var(--color-primary)"/>
          </marker>

          <marker
            id="arrowhead-waiting"
            markerWidth="10"
            markerHeight="7"
            refX="9"
            refY="3.5"
            orient="auto"
          >
            <polygon points="0 0, 10 3.5, 0 7" :fill="getStatusDotColor('waiting')"/>
          </marker>

          <linearGradient id="edge-running-gradient" x1="0%" y1="0%" x2="100%" y2="0%">
            <stop offset="0%" :stop-color="getStatusDotColor('running')" stop-opacity="0.7"/>
            <stop offset="100%" :stop-color="getRunningGradientEndColor()" stop-opacity="0.9"/>
          </linearGradient>
        </defs>
      </svg>
    </div>

    <div class="graph-legend">
      <div class="legend-item" v-for="status in statusTypes" :key="status">
        <span
          class="legend-dot"
          :style="{ background: getNodeColor(status) }"
        />
        <span
          class="legend-text"
        >
          {{ formatStatus(status) }}
        </span>
      </div>
    </div>
  </div>
</template>
<script>
export default {
  name: 'WorkflowGraph',

  props: {
    jobs: {
      type: Array,
      required: true,
    },
    currentJobId: {
      type: Number,
      required: true,
    },
  },

  data() {
    return {
      scale: 1,
      translateX: 0,
      translateY: 0,
      isDragging: false,
      dragStart: {x: 0, y: 0},
      lastMousePos: {x: 0, y: 0},
      animationFrameId: null,
      container: null,
      hoveredJobId: null,

      nodeHeight: 50,
      verticalSpacing: 120,
      margin: 40,
    };
  },

  computed: {
    nodeWidth() {
      let maxNameLength = 0;
      for (const job of this.jobs) {
        if (job.name.length > maxNameLength) {
          maxNameLength = job.name.length;
        }
      }
      return Math.min(Math.max(140, maxNameLength * 8), 180);
    },

    horizontalSpacing() {
      return this.nodeWidth + 20;
    },

    graphWidth() {
      if (this.jobsWithLayout.length === 0) return 800;
      let maxX = 0;
      for (const job of this.jobsWithLayout) {
        const x = job.x + this.nodeWidth;
        if (x > maxX) maxX = x;
      }
      return maxX + this.margin * 2;
    },

    graphHeight() {
      if (this.jobsWithLayout.length === 0) return 400;
      let maxY = 0;
      for (const job of this.jobsWithLayout) {
        const y = job.y + this.nodeHeight;
        if (y > maxY) maxY = y;
      }
      return maxY + this.margin * 2;
    },

    jobsWithLayout() {
      try {
        const levels = this.computeJobLevels(this.jobs);
        const currentHorizontalSpacing = this.horizontalSpacing;

        const jobsByLevel = [];
        let maxJobsPerLevel = 0;

        for (const job of this.jobs) {
          const level = levels.get(job.name) || levels.get(job.job_id) || 0;

          if (!jobsByLevel[level]) jobsByLevel[level] = [];
          jobsByLevel[level].push(job);

          if (jobsByLevel[level].length > maxJobsPerLevel) {
            maxJobsPerLevel = jobsByLevel[level].length;
          }
        }

        const result = [];
        for (const [levelIndex, levelJobs] of jobsByLevel.entries()) {
          if (!levelJobs || levelJobs.length === 0) continue;

          const levelWidth = (levelJobs.length - 1) * currentHorizontalSpacing;
          const startX = this.margin + (maxJobsPerLevel * currentHorizontalSpacing - levelWidth) / 2;

          for (const [jobIndex, job] of levelJobs.entries()) {
            result.push({
              ...job,
              x: startX + jobIndex * currentHorizontalSpacing,
              y: this.margin + levelIndex * this.verticalSpacing,
              level: levelIndex,
              index: this.jobs.findIndex((j) => j.id === job.id),
            });
          }
        }

        return result;
      } catch {
        const result = [];
        for (const [idx, job] of this.jobs.entries()) {
          result.push({
            ...job,
            x: this.margin + idx * (this.nodeWidth + 40),
            y: this.margin,
            level: 0,
            index: idx,
          });
        }
        return result;
      }
    },

    edges() {
      const edgesList = [];

      const jobsByJobId = new Map();
      for (const job of this.jobs) {
        if (job.job_id) {
          if (!jobsByJobId.has(job.job_id)) {
            jobsByJobId.set(job.job_id, []);
          }
          jobsByJobId.get(job.job_id).push(job);
        }
      }

      for (const job of this.jobs) {
        if (job.needs && job.needs.length > 0 && job.job_id) {
          for (const need of job.needs) {
            const targetJobs = jobsByJobId.get(need) || [];

            if (targetJobs.length > 0) {
              for (const targetJob of targetJobs) {
                edgesList.push({
                  from: targetJob.name,
                  to: job.name,
                  key: `${targetJob.id}-${job.id}`,
                });
              }
            } else {
              console.warn(`Job "${job.name}": need "${need}" not found`);
            }
          }
        }
      }

      return edgesList;
    },

    bezierEdges() {
      const bezierEdgesList = [];

      for (const edge of this.edges) {
        const fromNode = this.jobsWithLayout.find((j) => j.name === edge.from);
        const toNode = this.jobsWithLayout.find((j) => j.name === edge.to);

        if (!fromNode || !toNode) {
          continue;
        }

        const startX = fromNode.x + this.nodeWidth / 2;
        const startY = fromNode.y + this.nodeHeight;
        const endX = toNode.x + this.nodeWidth / 2;
        const endY = toNode.y;

        const levelDiff = toNode.level - fromNode.level;
        const curveStrength = 30 + Math.abs(levelDiff) * 15;

        const controlX1 = startX;
        const controlY1 = startY + curveStrength;
        const controlX2 = endX;
        const controlY2 = endY - curveStrength;

        const path = `M ${startX} ${startY} C ${controlX1} ${controlY1}, ${controlX2} ${controlY2}, ${endX} ${endY}`;

        bezierEdgesList.push({
          ...edge,
          path,
          fromNode,
          toNode,
        });
      }

      return bezierEdgesList;
    },

    graphMetrics() {
      let successCount = 0;
      for (const job of this.jobs) {
        const status = job.status.toLowerCase();
        if (status === 'success' || status === 'completed') {
          successCount++;
        }
      }

      const levels = new Map();
      for (const job of this.jobsWithLayout) {
        const count = levels.get(job.level) || 0;
        levels.set(job.level, count + 1);
      }
      let parallelism = 0;
      for (const count of levels.values()) {
        if (count > parallelism) parallelism = count;
      }

      return {
        successRate: `${((successCount / this.jobs.length) * 100).toFixed(0)}%`, parallelism,
      };
    },

    statusTypes() {
      const statuses = new Set();
      for (const job of this.jobs) {
        statuses.add(job.status);
      }
      return Array.from(statuses);
    },
  },

  mounted() {
    this.container = this.$refs.container;
  },

  methods: {
    zoomIn() {
      this.scale = Math.min(this.scale * 1.2, 3);
    },

    zoomOut() {
      this.scale = Math.max(this.scale * 0.8, 0.5);
    },

    resetView() {
      this.scale = 1;
      this.translateX = 0;
      this.translateY = 0;
    },

    handleWheel(event) {
      event.preventDefault();

      if (this.animationFrameId !== null) {
        cancelAnimationFrame(this.animationFrameId);
      }

      this.animationFrameId = requestAnimationFrame(() => {
        const delta = event.deltaY > 0 ? 0.9 : 1.1;
        this.scale = Math.min(Math.max(this.scale * delta, 0.5), 3);
      });
    },

    handleMouseDown(event) {
      if (event.button !== 0) return;
      event.preventDefault();

      this.isDragging = true;
      this.dragStart = {
        x: event.clientX - this.translateX,
        y: event.clientY - this.translateY,
      };
      this.lastMousePos = {x: event.clientX, y: event.clientY};

      if (this.container) {
        this.container.style.cursor = 'grabbing';
      }

      document.body.style.userSelect = 'none';
    },

    handleMouseMove(event) {
      if (!this.isDragging) return;

      if (this.animationFrameId !== null) {
        cancelAnimationFrame(this.animationFrameId);
      }

      this.animationFrameId = requestAnimationFrame(() => {
        const dx = event.clientX - this.lastMousePos.x;
        const dy = event.clientY - this.lastMousePos.y;

        this.translateX += dx;
        this.translateY += dy;

        this.lastMousePos = {x: event.clientX, y: event.clientY};
      });
    },

    handleMouseUp() {
      if (this.animationFrameId !== null) {
        cancelAnimationFrame(this.animationFrameId);
        this.animationFrameId = null;
      }

      this.isDragging = false;

      if (this.container) {
        this.container.style.cursor = 'grab';
      }

      document.body.style.userSelect = '';
    },

    handleMouseLeave() {
      this.handleMouseUp();
    },

    handleNodeMouseEnter(job) {
      this.hoveredJobId = job.id;
    },

    handleNodeMouseLeave() {
      this.hoveredJobId = null;
    },

    isEdgeHighlighted(edge) {
      if (!this.hoveredJobId) {
        return false;
      }

      const hoveredJob = this.jobsWithLayout.find((j) => j.id === this.hoveredJobId);
      if (!hoveredJob) {
        return false;
      }

      const highlighted = edge.from === hoveredJob.name || edge.to === hoveredJob.name;
      return highlighted;
    },

    getNodeColor(status) {
      const statusLower = status.toLowerCase();

      if (statusLower === 'success' || statusLower === 'completed') {
        return 'var(--color-green-darker, #1a7f37)';
      } else if (statusLower === 'failure') {
        return 'var(--color-red-darker, #cf222e)';
      } else if (statusLower === 'running') {
        return 'var(--color-yellow-darker, #bf8700)';
      } else if (statusLower === 'blocked') {
        return 'var(--color-purple, #8250df)';
      }

      return 'var(--color-text-light-3, #6e7781)';
    },

    getStatusDotColor(status) {
      const statusLower = status.toLowerCase();

      if (statusLower === 'success' || statusLower === 'completed') {
        return 'var(--color-green, #2ea043)';
      } else if (statusLower === 'failure') {
        return 'var(--color-red, #cf222e)';
      } else if (statusLower === 'running') {
        return 'var(--color-yellow, #e3b341)';
      }

      return 'var(--color-text-light-2, #848d97)';
    },

    getRunningGradientEndColor() {
      return 'var(--color-yellow-light, #ffd33d)';
    },

    getEdgeColor(edge) {
      if (!edge.fromNode || !edge.toNode) return 'var(--color-secondary, #d0d7de)';

      const fromStatus = edge.fromNode.status.toLowerCase();
      const toStatus = edge.toNode.status.toLowerCase();

      if (fromStatus === 'failure' || toStatus === 'failure') {
        return 'var(--color-red, #cf222e)';
      }

      if (fromStatus === 'running') {
        return 'var(--color-yellow, #e3b341)';
      }

      if (toStatus === 'running' && fromStatus === 'success') {
        return 'var(--color-primary, #0969da)';
      }

      if (fromStatus === 'success' && toStatus === 'success') {
        return 'var(--color-green, #2ea043)';
      }

      if (fromStatus === 'success' && (toStatus === 'waiting' || toStatus === 'blocked')) {
        return 'var(--color-primary-light, #54aeff)';
      }

      if (fromStatus === 'waiting' || fromStatus === 'blocked') {
        return 'var(--color-text-light-2, #848d97)';
      }

      if (fromStatus === 'cancelled' || toStatus === 'cancelled') {
        return 'var(--color-text-light-2, #848d97)';
      }

      return 'var(--color-secondary, #d0d7de)';
    },

    getDisplayName(name) {
      const maxChars = 26;
      if (name.length <= maxChars) return name;
      return `${name.substring(0, maxChars - 3)}...`;
    },

    formatStatus(status) {
      const statusMap = {
        success: 'Success',
        failure: 'Failed',
        running: 'Running',
        waiting: 'Waiting',
        cancelled: 'Cancelled',
        completed: 'Completed',
        blocked: 'Blocked',
      };
      return statusMap[status.toLowerCase()] || status;
    },

    getEdgeStyle(edge) {
      if (!edge.fromNode || !edge.toNode) {
        return {
          stroke: 'var(--color-secondary, #d0d7de)',
          strokeWidth: '2',
          opacity: '0.7',
        };
      }

      const fromStatus = edge.fromNode.status.toLowerCase();
      const toStatus = edge.toNode.status.toLowerCase();
      const isHighlighted = this.isEdgeHighlighted(edge);

      return {
        stroke: fromStatus === 'running' ? 'url(#edge-running-gradient)' : this.getEdgeColor(edge),
        strokeWidth: isHighlighted ? '3' : this.getStrokeWidth(fromStatus, toStatus),
        strokeDasharray: this.getDashArray(fromStatus, toStatus),
        opacity: isHighlighted ? 1 : this.getEdgeOpacity(fromStatus, toStatus),
        markerEnd: this.getMarkerEnd(fromStatus, toStatus),
        transition: 'all 0.2s ease',
      };
    },

    getStrokeWidth(fromStatus, toStatus) {
      if (fromStatus === 'running' || toStatus === 'running') return '3';
      if (fromStatus === 'failure' || toStatus === 'failure') return '2.5';
      return '2';
    },

    getDashArray(fromStatus, toStatus) {
      if (fromStatus === 'waiting' || toStatus === 'waiting') return '5,3';
      if (fromStatus === 'blocked') return '8,4';
      if (fromStatus === 'cancelled' || toStatus === 'cancelled') return '3,6';
      return 'none';
    },

    getEdgeOpacity(fromStatus, toStatus) {
      if (fromStatus === 'success' && toStatus === 'success') return 0.6;
      if (fromStatus === 'failure' || toStatus === 'failure') return 1;
      if (fromStatus === 'running' || toStatus === 'running') return 1;
      return 0.8;
    },

    getMarkerEnd(fromStatus, toStatus) {
      if (fromStatus === 'failure' || toStatus === 'failure') {
        return 'url(#arrowhead-failure)';
      }

      if (fromStatus === 'running') {
        return 'url(#arrowhead-running)';
      }

      if (fromStatus === 'success') {
        if (toStatus === 'running') return 'url(#arrowhead-ready)';
        if (toStatus === 'success' || toStatus === 'completed') return 'url(#arrowhead-success)';
      }

      if (fromStatus === 'waiting' || fromStatus === 'blocked') {
        return 'url(#arrowhead-waiting)';
      }

      return 'none';
    },

    getEdgeClass(edge) {
      if (!edge.fromNode || !edge.toNode) return '';

      const fromStatus = edge.fromNode.status.toLowerCase();
      const toStatus = edge.toNode.status.toLowerCase();

      const classes = [];

      if (fromStatus === 'running' || toStatus === 'running') {
        classes.push('running-edge');
      }

      if (fromStatus === 'success' && toStatus === 'success') {
        classes.push('success-edge');
      }

      if (fromStatus === 'failure' || toStatus === 'failure') {
        classes.push('failure-edge');
      }

      if (fromStatus === 'waiting' || toStatus === 'waiting') {
        classes.push('waiting-edge');
      }

      return classes.join(' ');
    },

    computeJobLevels(jobs) {
      const jobMap = new Map();
      for (const job of jobs) {
        jobMap.set(job.name, job);
        if (job.job_id) {
          jobMap.set(job.job_id, job);
        }
      }

      const levels = new Map();
      const visited = new Set();
      const recursionStack = new Set();
      const MAX_DEPTH = 100;

      const dfs = (jobNameOrId, depth = 0) => {
        if (depth > MAX_DEPTH) {
          console.error(`Max recursion depth (${MAX_DEPTH}) reached for: ${jobNameOrId}`);
          return 0;
        }

        if (recursionStack.has(jobNameOrId)) {
          console.error(`Cycle detected involving: ${jobNameOrId}`);
          return 0;
        }

        if (visited.has(jobNameOrId)) {
          return levels.get(jobNameOrId) || 0;
        }

        recursionStack.add(jobNameOrId);
        visited.add(jobNameOrId);

        const job = jobMap.get(jobNameOrId);
        if (!job) {
          recursionStack.delete(jobNameOrId);
          return 0;
        }

        if (!job.needs || job.needs.length === 0) {
          levels.set(job.name, 0);
          if (job.job_id && job.job_id !== job.name) {
            levels.set(job.job_id, 0);
          }
          recursionStack.delete(jobNameOrId);
          return 0;
        }

        let maxLevel = -1;
        for (const need of job.needs) {
          const needJob = jobMap.get(need);
          if (!needJob) {
            continue;
          }

          const needLevel = dfs(need, depth + 1);
          maxLevel = Math.max(maxLevel, needLevel);
        }

        const level = maxLevel + 1;
        levels.set(job.name, level);
        if (job.job_id && job.job_id !== job.name) {
          levels.set(job.job_id, level);
        }

        recursionStack.delete(jobNameOrId);
        return level;
      };

      for (const job of jobs) {
        if (!visited.has(job.name) && !visited.has(job.job_id)) {
          dfs(job.name);
        }
      }

      return levels;
    },

    onNodeClick(job, event) {
      if (job.index === this.currentJobId) {
        return;
      }

      const currentPath = window.location.pathname;
      const jobsIndex = currentPath.indexOf('/jobs/');

      if (jobsIndex !== -1) {
        const basePath = currentPath.substring(0, jobsIndex);
        const newJobUrl = `${basePath}/jobs/${job.index}`;

        const isCtrlClick = event?.ctrlKey || event?.metaKey;
        const isMiddleClick = event?.button === 1;
        const isNewTab = isCtrlClick || isMiddleClick;

        if (isNewTab) {
          window.open(newJobUrl, '_blank');
        } else {
          window.location.href = newJobUrl;
        }
      } else {
        const runMatch = currentPath.match(/\/runs\/(\d+)/);
        if (runMatch) {
          const runId = runMatch[1];
          const pathParts = currentPath.split(`/runs/${runId}`);
          const newJobUrl = `${pathParts[0]}/runs/${runId}/jobs/${job.id}`;

          if (event?.ctrlKey || event?.metaKey) {
            window.open(newJobUrl, '_blank');
          } else {
            window.location.href = newJobUrl;
          }
        }
      }
    },
  },
};
</script>

<style scoped>
.workflow-graph {
  margin: 20px 0;
  padding: 20px;
  background: var(--color-box-body);
  border-radius: 12px;
  border: 1px solid var(--color-secondary-alpha-20);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  position: relative;
}

.graph-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-secondary-alpha-20);
  gap: 15px;
  flex-wrap: wrap;
}

.graph-title {
  margin: 0;
  color: var(--color-text);
  font-size: 16px;
  font-weight: var(--font-weight-semibold);
  flex: 1;
  min-width: 200px;
}

.graph-stats {
  color: var(--color-text-light-2);
  font-size: 13px;
  padding: 4px 8px;
  background: var(--color-secondary-alpha-10);
  border-radius: 6px;
  white-space: nowrap;
}

.graph-metrics {
  color: var(--color-primary);
  font-weight: var(--font-weight-medium);
}

.graph-controls {
  display: flex;
  align-items: center;
  gap: 10px;
}

.control-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-body);
  border: 1px solid var(--color-secondary-alpha-30);
  border-radius: 4px;
  color: var(--color-text);
  cursor: pointer;
  transition: all 0.2s ease;
  padding: 0;
}

.control-btn:hover {
  background: var(--color-secondary-alpha-10);
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.control-icon {
  width: 16px;
  height: 16px;
}

.zoom-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--color-body);
  border: 1px solid var(--color-secondary-alpha-30);
  border-radius: 4px;
  padding: 0 4px;
}

.zoom-level {
  font-size: 12px;
  color: var(--color-text-light);
  min-width: 40px;
  text-align: center;
}

.graph-container {
  overflow: auto;
  padding: 10px;
  margin: -10px;
  border-radius: 8px;
  background: var(--color-body);
  cursor: grab;
  min-height: 300px;
  max-height: 600px;
  position: relative;
  -webkit-user-select: none;
  user-select: none;
}

.graph-container.dragging {
  cursor: grabbing;
}

.graph-svg {
  display: block;
  transition: transform 0.1s ease;
  will-change: transform;
}

.graph-svg path {
  transition: all 0.2s ease;
}

.highlighted-edge {
  stroke-width: 3 !important;
  opacity: 1 !important;
  stroke: var(--color-primary) !important;
}

.job-node-group {
  cursor: pointer;
  transition: all 0.2s ease;
  /* stylelint-disable-next-line function-no-unknown */
  --node-width: v-bind(nodeWidth + "px");
}

.job-node-group:hover .job-rect {
  filter: brightness(1.1);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 10;
}

.job-node-group.current-job .job-rect {
  filter: drop-shadow(0 0 8px color-mix(in srgb, var(--color-primary) 30%, transparent));
}

.job-name {
  max-width: calc(var(--node-width, 150px) - 50px);
  text-overflow: ellipsis;
  overflow: hidden;
  white-space: nowrap;
  pointer-events: none;
}

.job-status,
.job-duration,
.job-deps-label {
  user-select: none;
  pointer-events: none;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.7; transform: scale(1.1); }
}

.status-icon circle.pulse-dot {
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes shimmer {
  0% { background-position: -200px 0; }
  100% { background-position: calc(200px + 100%) 0; }
}

.running-background {
  animation: shimmer 2s infinite linear;
  background-size: 200px 100%;
}

@keyframes flowRunning {
  0% {
    stroke-dashoffset: 20;
    stroke-opacity: 0.7;
  }
  50% {
    stroke-opacity: 1;
  }
  100% {
    stroke-dashoffset: 0;
    stroke-opacity: 0.7;
  }
}

@keyframes pulseFailure {
  0%, 100% {
    stroke-width: 2.5;
    opacity: 0.7;
  }
  50% {
    stroke-width: 3;
    opacity: 1;
    filter: drop-shadow(0 0 4px color-mix(in srgb, var(--color-red) 50%, transparent));
  }
}

@keyframes shimmerEdge {
  0% {
    stroke-dashoffset: 20;
  }
  100% {
    stroke-dashoffset: 0;
  }
}

.running-edge {
  stroke-dasharray: 10, 5;
  animation: flowRunning 1s linear infinite;
}

.failure-edge {
  animation: pulseFailure 0.8s ease-in-out infinite;
}

.waiting-edge {
  stroke-dasharray: 5, 3;
  animation: shimmerEdge 2s linear infinite;
}

.success-edge {
  transition: stroke-width 0.3s ease, opacity 0.3s ease;
}

.success-edge:hover {
  stroke-width: 3;
  opacity: 1;
}

.progress-bar {
  animation: progressPulse 2s ease-in-out infinite;
}

@keyframes progressPulse {
  0%, 100% {
    opacity: 0.8;
  }
  50% {
    opacity: 1;
  }
}

.graph-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--color-secondary-alpha-20);
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.legend-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  display: inline-block;
}

.legend-text {
  color: var(--color-text-light);
  font-size: 12px;
  text-transform: capitalize;
}

@media (max-width: 768px) {
  .graph-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }

  .graph-controls {
    align-self: flex-end;
  }

  .graph-stats {
    font-size: 12px;
  }

  .workflow-graph {
    padding: 15px;
  }
}
</style>
