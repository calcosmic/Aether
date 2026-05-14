---
schema_version: "1.0"
id: oracle-tech-evaluation
template: tech-evaluation
kind: template
category: templates
title: Oracle Technology Evaluation Template
description: "Deep research structure for evaluating technologies, libraries, tools, or architectural options."
output_types: [tech-eval, technology-evaluation, tech-eval-example]
agent_roles: [oracle, architect, scout, queen]
task_types: [research, evaluation, architecture, integration, selection]
task_keywords: [evaluate, compare, versus, vs, library, framework, technology, tradeoff, adoption, ralf, recommendation, matrix]
workflow_triggers: [oracle, plan]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 5200
---
# Tech Evaluation: {{title}}

## Overview
<!-- 2-3 sentence summary of the technology area -->

## Alternatives Considered
<!-- List of options evaluated -->

## Evaluation Criteria
<!-- Criteria used for scoring -->

## Scoring Matrix
<!-- Table: alternative x criteria = score -->

## Recommendation
<!-- Clear recommendation with rationale -->
