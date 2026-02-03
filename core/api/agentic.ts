/**
 * Agentic API - Designed for AI Agent Interaction
 *
 * This API is designed from first principles for agentic use:
 * - Declarative intent over imperative commands
 * - Context-aware with conversation memory
 * - Confidence-first with epistemic metadata
 * - Self-documenting with discoverable capabilities
 * - Contribution-enabled (agents can give back)
 *
 * The API mirrors the PP hierarchy:
 *   /truth/*        → Ground queries in verified facts
 *   /understand/*   → Build comprehension, not just fetch data
 *   /beauty/*       → Explore coherent knowledge structures
 *   /justice/*      → Find the right information at right confidence
 *   /create/*       → Contribute new knowledge
 *
 * PP HIERARCHY:
 *   TRUTH: Every response grounded in verified facts
 *   UNDERSTANDING: Enable comprehension, not just data retrieval
 *   BEAUTY: Coherent, elegant knowledge structures
 *   JUSTICE: Right information at right confidence at right time
 *   CREATIVITY: Enable agents to contribute and extend knowledge
 */

import type { DID } from '../../agents/identity/did.js';
import {
  type PPScore,
  type PPMetadata,
  type PPSignal,
  type VirtueAssessment,
  calculatePPScore,
  createPPMetadata,
} from './philosophy.js';

// ============ Core Types ============

/**
 * Agent session - maintains context across queries
 */
export interface AgentSession {
  sessionId: string;
  agentDid: DID;
  context: SessionContext;
  queryHistory: QueryRecord[];
  virtueScores: VirtueAssessment[];
  startedAt: number;
  lastActivityAt: number;
}

export interface SessionContext {
  exploredDomains: string[];
  accessedFacts: string[];
  understandingGraph: UnderstandingNode[];
  openQuestions: string[];
  activePaths: CreativePath[];
}

export interface UnderstandingNode {
  factId: string;
  claim: string;
  confidence: number;
  accessedAt: number;
  usedFor: string[];
}

export interface CreativePath {
  pathId: string;
  description: string;
  startingFacts: string[];
  proposedExtensions: string[];
  status: 'exploring' | 'submitting' | 'verified' | 'abandoned';
}

export interface QueryRecord {
  queryId: string;
  query: AgentQuery;
  response: AgentResponse;
  timestamp: number;
  ppScores: PPScore;
}

// ============ Query Types ============

export interface AgentQuery {
  intent: QueryIntent;
  context?: QueryContext;
  confidence?: ConfidenceRequirements;
  output?: OutputPreferences;
}

export type QueryIntent =
  | UnderstandIntent
  | VerifyIntent
  | ExploreIntent
  | ConnectIntent
  | ContributeIntent
  | ChallengeIntent;

export interface UnderstandIntent {
  type: 'understand';
  topic: string;
  depth: 'surface' | 'working' | 'deep' | 'expert';
  purpose?: string;
}

export interface VerifyIntent {
  type: 'verify';
  claim: string;
  standard: 'factual' | 'logical' | 'empirical' | 'consensus';
}

export interface ExploreIntent {
  type: 'explore';
  from: string;
  direction: 'causes' | 'effects' | 'related' | 'contradictions' | 'extensions';
  breadth: number;
}

export interface ConnectIntent {
  type: 'connect';
  points: string[];
  connectionType: 'causal' | 'logical' | 'analogical' | 'any';
}

export interface ContributeIntent {
  type: 'contribute';
  contribution: Contribution;
}

export interface ChallengeIntent {
  type: 'challenge';
  targetFactId: string;
  evidence: string;
  challengeType: 'factual_error' | 'outdated' | 'incomplete' | 'misplaced';
}

export interface Contribution {
  type: 'claim' | 'evidence' | 'connection' | 'correction';
  content: string;
  evidence?: string[];
  domain?: string;
  selfConfidence?: number;
}

export interface QueryContext {
  useSessionContext: boolean;
  buildOn?: string[];
  focusDomains?: string[];
  exclude?: string[];
}

export interface ConfidenceRequirements {
  minimum: number;
  preferHighConfidence: boolean;
  includeContested: boolean;
  includeUnverified: boolean;
}

export interface OutputPreferences {
  format: 'structured' | 'narrative' | 'graph' | 'minimal';
  includeCitations: boolean;
  includePPMetadata: boolean;
  includeExtensions: boolean;
  maxFacts?: number;
}

// ============ Response Types ============

export interface AgentResponse {
  queryId: string;
  success: boolean;
  error?: AgentError;
  knowledge?: KnowledgePayload;
  pp: PPMetadata;
  understanding: UnderstandingGained;
  suggestions: QuerySuggestion[];
  cost: QueryCost;
}

export interface AgentError {
  code: string;
  message: string;
  signal?: PPSignal;
  resolution?: string;
}

export interface KnowledgePayload {
  facts: AgentFact[];
  relationships: FactRelationship[];
  narrative?: string;
  graph?: KnowledgeGraph;
}

export interface AgentFact {
  id: string;
  claim: string;
  domain: string;
  confidence: number;
  confidenceReasoning: string;
  citations: Citation[];
  verifiedAt: number;
  citationCount: number;
  ppScore: PPScore;
  signals: PPSignal[];
}

export interface Citation {
  sourceId: string;
  relationship: 'supports' | 'derives_from' | 'contradicts' | 'extends';
  strength: number;
  explanation: string;
}

export interface FactRelationship {
  from: string;
  to: string;
  type: 'supports' | 'contradicts' | 'extends' | 'specializes' | 'generalizes';
  strength: number;
}

export interface KnowledgeGraph {
  nodes: GraphNode[];
  edges: GraphEdge[];
  clusters: GraphCluster[];
}

export interface GraphNode {
  id: string;
  label: string;
  type: 'fact' | 'domain' | 'concept';
  confidence: number;
  size: number;
}

export interface GraphEdge {
  source: string;
  target: string;
  type: string;
  weight: number;
}

export interface GraphCluster {
  id: string;
  label: string;
  nodes: string[];
  coherence: number;
}

export interface UnderstandingGained {
  newUnderstanding: string[];
  connections: string[];
  gaps: string[];
  depthAchieved: 'surface' | 'working' | 'deep' | 'expert';
}

export interface QuerySuggestion {
  query: Partial<AgentQuery>;
  reasoning: string;
  ppDimension: 'truth' | 'understanding' | 'beauty' | 'justice' | 'creativity';
  estimatedValue: number;
}

export interface QueryCost {
  total: bigint;
  breakdown: {
    baseFee: bigint;
    confidencePremium: bigint;
    depthPremium: bigint;
    creativityBonus: bigint;
  };
  researchContribution: bigint;
  knowledgeRewards: bigint;
}

// ============ API Implementation ============

/**
 * Agentic API - Main interface for AI agents
 */
export class AgenticAPI {
  private sessions: Map<string, AgentSession> = new Map();

  async createSession(agentDid: DID): Promise<AgentSession> {
    const sessionId = `session:${agentDid}:${Date.now()}`;

    const session: AgentSession = {
      sessionId,
      agentDid,
      context: {
        exploredDomains: [],
        accessedFacts: [],
        understandingGraph: [],
        openQuestions: [],
        activePaths: [],
      },
      queryHistory: [],
      virtueScores: [],
      startedAt: Date.now(),
      lastActivityAt: Date.now(),
    };

    this.sessions.set(sessionId, session);
    return session;
  }

  async query(sessionId: string, query: AgentQuery): Promise<AgentResponse> {
    const session = this.sessions.get(sessionId);
    if (!session) {
      return this.errorResponse('SESSION_NOT_FOUND', 'Session not found');
    }

    switch (query.intent.type) {
      case 'understand':
        return this.handleUnderstand(session, query);
      case 'verify':
        return this.handleVerify(session, query);
      case 'explore':
        return this.handleExplore(session, query);
      case 'connect':
        return this.handleConnect(session, query);
      case 'contribute':
        return this.handleContribute(session, query);
      case 'challenge':
        return this.handleChallenge(session, query);
      default:
        return this.errorResponse('UNKNOWN_INTENT', 'Unknown query intent');
    }
  }

  private async handleUnderstand(
    session: AgentSession,
    query: AgentQuery
  ): Promise<AgentResponse> {
    const intent = query.intent as UnderstandIntent;
    const depthMultiplier = { surface: 1, working: 3, deep: 7, expert: 15 }[intent.depth];

    const facts = this.mockFactsForTopic(intent.topic, depthMultiplier);
    const relationships = this.mockRelationships(facts);

    const pp = createPPMetadata(
      { confidence: 0.85, domain: intent.topic, citationCount: facts.length },
      0.8,
      false,
      facts.map((f, i) => ({
        factId: f.id,
        claim: f.claim,
        confidence: f.confidence,
        depth: i,
        relationship: 'supports' as const,
      }))
    );

    const understanding: UnderstandingGained = {
      newUnderstanding: facts.map(f => `Understand: ${f.claim}`),
      connections: relationships.map(r => `${r.from} ${r.type} ${r.to}`),
      gaps: this.identifyGaps(intent.topic, facts),
      depthAchieved: intent.depth,
    };

    const suggestions = this.generateSuggestions(intent, understanding);

    // Update session context
    session.context.exploredDomains.push(intent.topic);
    session.context.accessedFacts.push(...facts.map(f => f.id));
    session.lastActivityAt = Date.now();

    return {
      queryId: `query:${Date.now()}`,
      success: true,
      knowledge: { facts, relationships, narrative: this.generateNarrative(intent.topic, facts) },
      pp,
      understanding,
      suggestions,
      cost: this.calculateCost(facts.length, intent.depth),
    };
  }

  private async handleVerify(
    session: AgentSession,
    query: AgentQuery
  ): Promise<AgentResponse> {
    const intent = query.intent as VerifyIntent;
    const verification = this.mockVerification(intent.claim, intent.standard);

    session.lastActivityAt = Date.now();

    const pp = createPPMetadata(
      { confidence: verification.confidence, domain: 'verification', citationCount: verification.supportingFacts.length },
      0.9,
      false,
      verification.supportingFacts.map((f, i) => ({
        factId: f,
        claim: `Supporting fact ${i + 1}`,
        confidence: verification.confidence,
        depth: i,
        relationship: 'supports' as const,
      }))
    );

    return {
      queryId: `query:${Date.now()}`,
      success: true,
      knowledge: {
        facts: [{
          id: `verification:${Date.now()}`,
          claim: intent.claim,
          domain: 'verification',
          confidence: verification.confidence,
          confidenceReasoning: verification.reasoning,
          citations: verification.supportingFacts.map(f => ({
            sourceId: f,
            relationship: 'supports' as const,
            strength: 0.8,
            explanation: 'Supports this claim',
          })),
          verifiedAt: Date.now(),
          citationCount: 0,
          ppScore: calculatePPScore({ truth: verification.confidence, understanding: 0.7, beauty: 0.8, justice: 0.9, creativity: 0.2 }),
          signals: [],
        }],
        relationships: [],
      },
      pp,
      understanding: {
        newUnderstanding: [`Verified: "${intent.claim}" with ${(verification.confidence * 100).toFixed(0)}% confidence`],
        connections: verification.supportingFacts.map(f => `Supported by ${f}`),
        gaps: verification.confidence < 0.8 ? ['Additional verification may strengthen confidence'] : [],
        depthAchieved: 'working',
      },
      suggestions: verification.confidence < 0.9 ? [{
        query: { intent: { type: 'explore', from: intent.claim, direction: 'related', breadth: 5 } },
        reasoning: 'Explore related facts to increase confidence',
        ppDimension: 'truth',
        estimatedValue: 0.2,
      }] : [],
      cost: this.calculateCost(verification.supportingFacts.length, 'working'),
    };
  }

  private async handleExplore(
    session: AgentSession,
    query: AgentQuery
  ): Promise<AgentResponse> {
    const intent = query.intent as ExploreIntent;
    const explored = this.mockExploration(intent.from, intent.direction, intent.breadth);

    session.lastActivityAt = Date.now();

    const pp = createPPMetadata(
      { confidence: 0.75, domain: 'exploration', citationCount: explored.facts.length },
      0.7,
      true,
      explored.facts.map((f, i) => ({
        factId: f.id,
        claim: f.claim,
        confidence: f.confidence,
        depth: i,
        relationship: this.directionToRelationship(intent.direction),
      }))
    );

    if (explored.novelPaths.length > 0) {
      session.context.activePaths.push(...explored.novelPaths.map(p => ({
        pathId: `path:${Date.now()}:${Math.random().toString(36).slice(2)}`,
        description: p,
        startingFacts: [intent.from],
        proposedExtensions: [],
        status: 'exploring' as const,
      })));
    }

    return {
      queryId: `query:${Date.now()}`,
      success: true,
      knowledge: {
        facts: explored.facts,
        relationships: explored.relationships,
        graph: this.buildGraph(explored.facts, explored.relationships),
      },
      pp,
      understanding: {
        newUnderstanding: explored.facts.map(f => f.claim),
        connections: explored.relationships.map(r => `${r.from} → ${r.to}`),
        gaps: explored.gaps,
        depthAchieved: 'working',
      },
      suggestions: explored.novelPaths.map(path => ({
        query: { intent: { type: 'contribute', contribution: { type: 'claim', content: path } } },
        reasoning: 'Novel path discovered - consider contributing',
        ppDimension: 'creativity',
        estimatedValue: 0.5,
      })),
      cost: this.calculateCost(explored.facts.length, 'working'),
    };
  }

  private async handleConnect(
    session: AgentSession,
    query: AgentQuery
  ): Promise<AgentResponse> {
    const intent = query.intent as ConnectIntent;
    const connections = this.mockConnections(intent.points, intent.connectionType);

    session.lastActivityAt = Date.now();

    const pp = createPPMetadata(
      { confidence: connections.confidence, domain: 'connections', citationCount: connections.path.length },
      0.85,
      connections.isNovel,
      connections.path.map((p, i) => ({
        factId: p.factId,
        claim: p.claim,
        confidence: p.confidence,
        depth: i,
        relationship: 'supports' as const,
      }))
    );

    return {
      queryId: `query:${Date.now()}`,
      success: connections.found,
      error: connections.found ? undefined : {
        code: 'NO_CONNECTION',
        message: `Could not find ${intent.connectionType} connection between points`,
        signal: {
          type: 'stagnation',
          description: 'No existing connection found',
          unchartedTerritory: `Connection between ${intent.points.join(' and ')}`,
          creativePath: 'Consider contributing this connection as new knowledge',
        },
      },
      knowledge: connections.found ? {
        facts: connections.path.map(p => ({
          id: p.factId,
          claim: p.claim,
          domain: 'connections',
          confidence: p.confidence,
          confidenceReasoning: 'Part of connection chain',
          citations: [],
          verifiedAt: Date.now(),
          citationCount: 0,
          ppScore: calculatePPScore({ truth: p.confidence, understanding: 0.8, beauty: 0.7, justice: 0.8, creativity: connections.isNovel ? 0.9 : 0.3 }),
          signals: [],
        })),
        relationships: connections.path.slice(0, -1).map((p, i) => ({
          from: p.factId,
          to: connections.path[i + 1].factId,
          type: 'supports' as const,
          strength: 0.8,
        })),
      } : undefined,
      pp,
      understanding: {
        newUnderstanding: connections.found ? [`Connection found: ${intent.points.join(' → ')}`] : ['No direct connection exists'],
        connections: connections.path.map(p => p.claim),
        gaps: connections.found ? [] : [`Gap between ${intent.points.join(' and ')}`],
        depthAchieved: 'deep',
      },
      suggestions: connections.isNovel ? [{
        query: { intent: { type: 'contribute', contribution: { type: 'connection', content: `Connection: ${intent.points.join(' → ')}` } } },
        reasoning: 'Novel connection discovered - contribute to knowledge base',
        ppDimension: 'creativity',
        estimatedValue: 0.7,
      }] : [],
      cost: this.calculateCost(connections.path.length, 'deep'),
    };
  }

  private async handleContribute(
    session: AgentSession,
    query: AgentQuery
  ): Promise<AgentResponse> {
    const intent = query.intent as ContributeIntent;
    const validation = this.validateContribution(intent.contribution);

    session.lastActivityAt = Date.now();

    const pp = createPPMetadata(
      { confidence: validation.estimatedConfidence, domain: intent.contribution.domain || 'general', citationCount: 0 },
      0.6,
      true,
      []
    );

    if (!validation.valid) {
      return {
        queryId: `query:${Date.now()}`,
        success: false,
        error: {
          code: 'INVALID_CONTRIBUTION',
          message: validation.reason || 'Contribution validation failed',
          signal: {
            type: 'ugliness',
            description: 'Contribution needs improvement',
            understandingNeeded: validation.improvements?.join(', ') || 'Better evidence',
            suggestion: 'Strengthen evidence or clarify claim',
          },
        },
        pp,
        understanding: { newUnderstanding: [], connections: [], gaps: validation.improvements || [], depthAchieved: 'surface' },
        suggestions: [{
          query: { intent: { type: 'explore', from: intent.contribution.content, direction: 'related', breadth: 3 } },
          reasoning: 'Explore related facts to strengthen contribution',
          ppDimension: 'truth',
          estimatedValue: 0.4,
        }],
        cost: this.calculateCost(0, 'surface'),
      };
    }

    const submission = this.mockSubmitContribution(intent.contribution);

    return {
      queryId: `query:${Date.now()}`,
      success: true,
      knowledge: {
        facts: [{
          id: submission.claimId,
          claim: intent.contribution.content,
          domain: intent.contribution.domain || 'general',
          confidence: submission.initialConfidence,
          confidenceReasoning: 'Pending verification',
          citations: [],
          verifiedAt: 0,
          citationCount: 0,
          ppScore: calculatePPScore({ truth: submission.initialConfidence, understanding: 0.5, beauty: 0.6, justice: 0.7, creativity: 0.9 }),
          signals: [],
        }],
        relationships: [],
      },
      pp,
      understanding: { newUnderstanding: ['Contribution submitted for verification'], connections: [], gaps: [], depthAchieved: 'working' },
      suggestions: [{
        query: { intent: { type: 'verify', claim: intent.contribution.content, standard: 'factual' } },
        reasoning: 'Monitor verification progress',
        ppDimension: 'truth',
        estimatedValue: 0.3,
      }],
      cost: {
        total: submission.stake,
        breakdown: { baseFee: submission.stake, confidencePremium: 0n, depthPremium: 0n, creativityBonus: -submission.stake / 10n },
        researchContribution: submission.stake * 22n / 100n,
        knowledgeRewards: 0n,
      },
    };
  }

  private async handleChallenge(
    session: AgentSession,
    query: AgentQuery
  ): Promise<AgentResponse> {
    const intent = query.intent as ChallengeIntent;
    const challenge = this.mockSubmitChallenge(intent);

    session.virtueScores.push({ virtue: 'courage', score: 1.0, evidence: `Challenged fact ${intent.targetFactId}` });
    session.lastActivityAt = Date.now();

    const pp = createPPMetadata({ confidence: 0.5, domain: 'challenge', citationCount: 1 }, 0.5, false, []);

    return {
      queryId: `query:${Date.now()}`,
      success: true,
      knowledge: {
        facts: [{
          id: challenge.challengeId,
          claim: `Challenge to: ${intent.targetFactId}`,
          domain: 'challenges',
          confidence: 0.5,
          confidenceReasoning: 'Challenge pending resolution',
          citations: [{ sourceId: intent.targetFactId, relationship: 'contradicts', strength: 0.8, explanation: intent.evidence }],
          verifiedAt: 0,
          citationCount: 0,
          ppScore: calculatePPScore({ truth: 0.5, understanding: 0.6, beauty: 0.5, justice: 0.8, creativity: 0.4 }),
          signals: [],
        }],
        relationships: [{ from: challenge.challengeId, to: intent.targetFactId, type: 'contradicts', strength: 0.8 }],
      },
      pp,
      understanding: { newUnderstanding: ['Challenge submitted'], connections: [`Contradicts ${intent.targetFactId}`], gaps: [], depthAchieved: 'working' },
      suggestions: [],
      cost: {
        total: challenge.stake,
        breakdown: { baseFee: challenge.stake, confidencePremium: 0n, depthPremium: 0n, creativityBonus: 0n },
        researchContribution: challenge.stake * 22n / 100n,
        knowledgeRewards: 0n,
      },
    };
  }

  // ============ Helper Methods ============

  private errorResponse(code: string, message: string): AgentResponse {
    return {
      queryId: `error:${Date.now()}`,
      success: false,
      error: { code, message },
      pp: createPPMetadata({ confidence: 0, domain: 'error', citationCount: 0 }, 0, false, []),
      understanding: { newUnderstanding: [], connections: [], gaps: [message], depthAchieved: 'surface' },
      suggestions: [],
      cost: { total: 0n, breakdown: { baseFee: 0n, confidencePremium: 0n, depthPremium: 0n, creativityBonus: 0n }, researchContribution: 0n, knowledgeRewards: 0n },
    };
  }

  private mockFactsForTopic(topic: string, count: number): AgentFact[] {
    return Array.from({ length: Math.min(count, 10) }, (_, i) => ({
      id: `fact:${topic}:${i}`,
      claim: `Fact ${i + 1} about ${topic}`,
      domain: topic,
      confidence: 0.7 + Math.random() * 0.25,
      confidenceReasoning: 'Based on multiple verifications',
      citations: [],
      verifiedAt: Date.now() - i * 86400000,
      citationCount: Math.floor(Math.random() * 20),
      ppScore: calculatePPScore({ truth: 0.8, understanding: 0.7, beauty: 0.75, justice: 0.8, creativity: 0.3 }),
      signals: [],
    }));
  }

  private mockRelationships(facts: AgentFact[]): FactRelationship[] {
    if (facts.length < 2) return [];
    return facts.slice(1).map((f, i) => ({ from: facts[i].id, to: f.id, type: 'supports' as const, strength: 0.7 }));
  }

  private identifyGaps(topic: string, facts: AgentFact[]): string[] {
    return facts.length < 3 ? [`Limited knowledge about ${topic} - consider exploring further`] : [];
  }

  private generateSuggestions(intent: UnderstandIntent, understanding: UnderstandingGained): QuerySuggestion[] {
    const suggestions: QuerySuggestion[] = [];
    if (understanding.gaps.length > 0) {
      suggestions.push({ query: { intent: { type: 'explore', from: intent.topic, direction: 'related', breadth: 5 } }, reasoning: 'Fill gaps in understanding', ppDimension: 'understanding', estimatedValue: 0.4 });
    }
    if (intent.depth !== 'expert') {
      suggestions.push({ query: { intent: { type: 'understand', topic: intent.topic, depth: 'deep' } }, reasoning: 'Deepen understanding', ppDimension: 'understanding', estimatedValue: 0.3 });
    }
    return suggestions;
  }

  private generateNarrative(topic: string, facts: AgentFact[]): string {
    return `Understanding ${topic}: ${facts.map(f => f.claim).join('. ')}`;
  }

  private calculateCost(factCount: number, depth: 'surface' | 'working' | 'deep' | 'expert'): QueryCost {
    const depthMultiplier = { surface: 1, working: 2, deep: 4, expert: 8 }[depth];
    const baseFee = BigInt(factCount) * 1000000000000000n;
    const depthPremium = baseFee * BigInt(depthMultiplier - 1);
    const total = baseFee + depthPremium;
    return {
      total,
      breakdown: { baseFee, confidencePremium: 0n, depthPremium, creativityBonus: 0n },
      researchContribution: total * 22n / 100n,
      knowledgeRewards: total * 30n / 100n,
    };
  }

  private mockVerification(claim: string, _standard: string): { confidence: number; reasoning: string; supportingFacts: string[] } {
    return { confidence: 0.75 + Math.random() * 0.2, reasoning: `Verified claim: "${claim}"`, supportingFacts: [`fact:support:${Date.now()}:1`, `fact:support:${Date.now()}:2`] };
  }

  private mockExploration(from: string, direction: string, breadth: number): { facts: AgentFact[]; relationships: FactRelationship[]; gaps: string[]; novelPaths: string[] } {
    const facts = this.mockFactsForTopic(`${from}:${direction}`, breadth);
    return { facts, relationships: this.mockRelationships(facts), gaps: [], novelPaths: Math.random() > 0.7 ? [`Novel path from ${from} via ${direction}`] : [] };
  }

  private directionToRelationship(direction: string): 'supports' | 'extends' | 'contradicts' | 'specializes' | 'generalizes' {
    const map: Record<string, 'supports' | 'extends' | 'contradicts' | 'specializes' | 'generalizes'> = { causes: 'supports', effects: 'extends', related: 'supports', contradictions: 'contradicts', extensions: 'extends' };
    return map[direction] || 'supports';
  }

  private mockConnections(points: string[], _type: string): { found: boolean; confidence: number; path: Array<{ factId: string; claim: string; confidence: number }>; isNovel: boolean } {
    const found = Math.random() > 0.3;
    return { found, confidence: found ? 0.7 + Math.random() * 0.25 : 0, path: found ? points.map((p, i) => ({ factId: `fact:path:${i}`, claim: `Step ${i + 1}: ${p}`, confidence: 0.8 })) : [], isNovel: found && Math.random() > 0.7 };
  }

  private buildGraph(facts: AgentFact[], relationships: FactRelationship[]): KnowledgeGraph {
    return {
      nodes: facts.map(f => ({ id: f.id, label: f.claim, type: 'fact' as const, confidence: f.confidence, size: f.citationCount + 1 })),
      edges: relationships.map(r => ({ source: r.from, target: r.to, type: r.type, weight: r.strength })),
      clusters: [{ id: 'main', label: 'Main Cluster', nodes: facts.map(f => f.id), coherence: 0.8 }],
    };
  }

  private validateContribution(contribution: Contribution): { valid: boolean; reason?: string; estimatedConfidence: number; improvements?: string[] } {
    if (!contribution.content || contribution.content.length < 10) {
      return { valid: false, reason: 'Claim too short', estimatedConfidence: 0, improvements: ['Provide more detailed claim'] };
    }
    if (!contribution.evidence || contribution.evidence.length === 0) {
      return { valid: false, reason: 'No evidence provided', estimatedConfidence: 0.2, improvements: ['Add supporting evidence'] };
    }
    return { valid: true, estimatedConfidence: 0.5 + (contribution.evidence.length * 0.1) };
  }

  private mockSubmitContribution(_contribution: Contribution): { claimId: string; stake: bigint; initialConfidence: number } {
    return { claimId: `claim:${Date.now()}`, stake: 100n * 10n ** 18n, initialConfidence: 0.5 };
  }

  private mockSubmitChallenge(_intent: ChallengeIntent): { challengeId: string; stake: bigint } {
    return { challengeId: `challenge:${Date.now()}`, stake: 50n * 10n ** 18n };
  }
}
