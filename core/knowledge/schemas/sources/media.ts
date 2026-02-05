/**
 * Media Source Schemas
 *
 * Schemas for news articles, press releases, and media content.
 *
 * Media sources require special handling:
 * - Content may change or disappear
 * - Requires archival verification
 * - Attribution and sourcing important
 * - Bias and editorial stance should be noted
 */

import type { ClaimSchema } from "../types.js";

// ============ News Article Schema ============

/**
 * News Article Schema
 *
 * A news article from a media outlet.
 */
export const NEWS_ARTICLE_SCHEMA: ClaimSchema = {
  id: "news_article_v1",
  version: 1,
  name: "News Article",
  description: "A news article with extractable claims",
  domain: "media.news",

  timing: {
    commitPhaseBlocks: 8,
    revealPhaseBlocks: 6,
    totalRoundBlocks: 16,
    minAnalysisTimeMs: 8000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Article Identification ============
      url: {
        type: "string",
        description: "Original article URL",
      },
      archiveUrl: {
        type: "string",
        description: "Archive.org or perma.cc URL (required for verification)",
      },
      contentHash: {
        type: "string",
        description: "SHA-256 hash of article content",
      },

      // ============ Publication Details ============
      headline: {
        type: "string",
        description: "Article headline",
      },
      subheadline: {
        type: "string",
        description: "Subheadline or deck",
      },
      outlet: {
        type: "string",
        description: "Publishing outlet name",
      },
      outletType: {
        type: "string",
        enum: [
          "wire_service", // AP, Reuters, AFP
          "national_newspaper", // NYT, WSJ, Guardian
          "local_newspaper",
          "broadcast_network", // CNN, BBC, NBC
          "digital_native", // Axios, Politico, Vox
          "trade_publication", // Industry-specific
          "tabloid",
          "opinion_outlet",
          "state_media",
          "independent",
          "blog",
          "other",
        ],
        description: "Type of media outlet",
      },
      publishedDate: {
        type: "string",
        description: "Publication date (ISO 8601)",
      },
      lastModified: {
        type: "string",
        description: "Last modification date (if updated)",
      },

      // ============ Authorship ============
      authors: {
        type: "array",
        items: {
          type: "object",
          properties: {
            name: { type: "string" },
            role: { type: "string" }, // reporter, editor, contributor
            email: { type: "string" },
            twitter: { type: "string" },
          },
        },
        description: "Article authors",
      },
      isOpinion: {
        type: "boolean",
        description: "Is this an opinion/editorial piece?",
      },
      isAnalysis: {
        type: "boolean",
        description: "Is this analysis vs straight news?",
      },

      // ============ Content Classification ============
      section: {
        type: "string",
        description: "Section/category (politics, tech, business, etc.)",
      },
      topics: {
        type: "array",
        items: { type: "string" },
        description: "Topics covered",
      },
      geography: {
        type: "array",
        items: { type: "string" },
        description: "Geographic relevance",
      },

      // ============ Sourcing ============
      namedSources: {
        type: "array",
        items: {
          type: "object",
          properties: {
            name: { type: "string" },
            title: { type: "string" },
            organization: { type: "string" },
            quoteCount: { type: "number" },
          },
        },
        description: "Named sources in the article",
      },
      anonymousSourceCount: {
        type: "number",
        description: "Number of anonymous sources cited",
      },
      documentsReferenced: {
        type: "array",
        items: { type: "string" },
        description: "Documents referenced in article",
      },

      // ============ Corrections ============
      hasCorrection: {
        type: "boolean",
        description: "Has article been corrected?",
      },
      corrections: {
        type: "array",
        items: {
          type: "object",
          properties: {
            date: { type: "string" },
            description: { type: "string" },
          },
        },
        description: "Correction history",
      },

      // ============ Extracted Claims ============
      extractedClaims: {
        type: "array",
        items: {
          type: "object",
          properties: {
            claimText: { type: "string" },
            claimType: {
              type: "string",
              enum: ["fact", "quote", "attribution", "analysis", "prediction"],
            },
            location: { type: "string" },
            sourcedTo: { type: "string" },
            verifiable: { type: "boolean" },
          },
        },
        description: "Claims extracted from article",
      },
    },
    required: ["url", "archiveUrl", "headline", "outlet", "publishedDate"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["archived_web"],
  minReferences: 1,

  active: true,
};

// ============ Press Release Schema ============

/**
 * Press Release Schema
 *
 * An official press release or announcement.
 */
export const PRESS_RELEASE_SCHEMA: ClaimSchema = {
  id: "press_release_v1",
  version: 1,
  name: "Press Release",
  description: "An official press release or corporate announcement",
  domain: "media.press_releases",

  timing: {
    commitPhaseBlocks: 6,
    revealPhaseBlocks: 4,
    totalRoundBlocks: 12,
    minAnalysisTimeMs: 6000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Identification ============
      url: {
        type: "string",
        description: "Original press release URL",
      },
      archiveUrl: {
        type: "string",
        description: "Archive URL",
      },
      wireService: {
        type: "string",
        enum: [
          "pr_newswire",
          "business_wire",
          "globe_newswire",
          "cision",
          "direct",
          "other",
        ],
        description: "Distribution wire service",
      },
      wireId: {
        type: "string",
        description: "Wire service release ID",
      },

      // ============ Source ============
      organization: {
        type: "string",
        description: "Issuing organization",
      },
      organizationType: {
        type: "string",
        enum: [
          "corporation",
          "government",
          "nonprofit",
          "academic",
          "political",
          "other",
        ],
        description: "Type of organization",
      },
      ticker: {
        type: "string",
        description: "Stock ticker (if public company)",
      },

      // ============ Content ============
      headline: {
        type: "string",
        description: "Press release headline",
      },
      releaseDate: {
        type: "string",
        description: "Release date/time",
      },
      releaseType: {
        type: "string",
        enum: [
          "earnings",
          "product_launch",
          "partnership",
          "acquisition",
          "executive_change",
          "legal",
          "regulatory",
          "event",
          "research",
          "general",
        ],
        description: "Type of announcement",
      },

      // ============ Contacts ============
      mediaContacts: {
        type: "array",
        items: {
          type: "object",
          properties: {
            name: { type: "string" },
            title: { type: "string" },
            email: { type: "string" },
            phone: { type: "string" },
          },
        },
        description: "Media contact information",
      },

      // ============ Quotes ============
      quotes: {
        type: "array",
        items: {
          type: "object",
          properties: {
            speaker: { type: "string" },
            title: { type: "string" },
            quote: { type: "string" },
          },
        },
        description: "Quotes in the release",
      },

      // ============ Forward-Looking Statements ============
      containsForwardLooking: {
        type: "boolean",
        description: "Contains forward-looking statements?",
      },
      safeHarborStatement: {
        type: "boolean",
        description: "Includes safe harbor disclaimer?",
      },
    },
    required: ["url", "organization", "headline", "releaseDate"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["archived_web"],
  minReferences: 1,

  active: true,
};

// ============ Quote Attribution Schema ============

/**
 * Quote Attribution Schema
 *
 * A specific quote attributed to a person.
 */
export const QUOTE_ATTRIBUTION_SCHEMA: ClaimSchema = {
  id: "quote_attribution_v1",
  version: 1,
  name: "Quote Attribution",
  description: "A quote attributed to a specific person",
  domain: "media.quotes",

  timing: {
    commitPhaseBlocks: 6,
    revealPhaseBlocks: 4,
    totalRoundBlocks: 12,
    minAnalysisTimeMs: 5000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ The Quote ============
      quote: {
        type: "string",
        description: "The exact quote",
      },
      quoteHash: {
        type: "string",
        description: "Hash of normalized quote text",
      },
      isDirectQuote: {
        type: "boolean",
        description: "Is this a direct quote (vs paraphrase)?",
      },
      language: {
        type: "string",
        description: "Original language of quote",
      },
      isTranslation: {
        type: "boolean",
        description: "Is this a translation?",
      },

      // ============ Attribution ============
      speaker: {
        type: "string",
        description: "Person being quoted",
      },
      speakerTitle: {
        type: "string",
        description: "Speaker title/role at time of quote",
      },
      speakerOrganization: {
        type: "string",
        description: "Speaker organization",
      },

      // ============ Context ============
      context: {
        type: "string",
        description: "Context in which quote was given",
      },
      venue: {
        type: "string",
        description: "Where the quote was given",
      },
      date: {
        type: "string",
        description: "Date of quote",
      },

      // ============ Source ============
      sourceType: {
        type: "string",
        enum: [
          "interview",
          "speech",
          "press_conference",
          "document",
          "social_media",
          "broadcast",
          "other",
        ],
        description: "How the quote was captured",
      },
      sourceUrl: {
        type: "string",
        description: "URL to source",
      },
      archiveUrl: {
        type: "string",
        description: "Archive URL",
      },
      videoTimestamp: {
        type: "string",
        description: "Timestamp in video (if applicable)",
      },

      // ============ Verification ============
      multipleSourcesConfirm: {
        type: "boolean",
        description: "Do multiple independent sources confirm?",
      },
      speakerConfirmed: {
        type: "boolean",
        description: "Has the speaker confirmed the quote?",
      },
      speakerDenied: {
        type: "boolean",
        description: "Has the speaker denied the quote?",
      },
    },
    required: ["quote", "speaker", "sourceType"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["archived_web", "attestation"],
  minReferences: 1,

  active: true,
};

// ============ Social Media Post Schema ============

/**
 * Social Media Post Schema
 *
 * A post from a social media platform.
 */
export const SOCIAL_MEDIA_POST_SCHEMA: ClaimSchema = {
  id: "social_media_post_v1",
  version: 1,
  name: "Social Media Post",
  description: "A social media post with verifiable claims",
  domain: "media.social",

  timing: {
    commitPhaseBlocks: 4,
    revealPhaseBlocks: 4,
    totalRoundBlocks: 10,
    minAnalysisTimeMs: 4000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Platform & Post ============
      platform: {
        type: "string",
        enum: [
          "twitter",
          "facebook",
          "linkedin",
          "instagram",
          "tiktok",
          "youtube",
          "reddit",
          "telegram",
          "discord",
          "other",
        ],
        description: "Social media platform",
      },
      postUrl: {
        type: "string",
        description: "Original post URL",
      },
      archiveUrl: {
        type: "string",
        description: "Archive URL (critical - posts often deleted)",
      },
      postId: {
        type: "string",
        description: "Platform-specific post ID",
      },

      // ============ Account ============
      accountHandle: {
        type: "string",
        description: "Account username/handle",
      },
      accountName: {
        type: "string",
        description: "Display name",
      },
      accountVerified: {
        type: "boolean",
        description: "Is account verified by platform?",
      },
      accountType: {
        type: "string",
        enum: [
          "individual",
          "organization",
          "government",
          "media",
          "bot",
          "unknown",
        ],
        description: "Type of account",
      },
      followerCount: {
        type: "number",
        description: "Follower count at time of post",
      },

      // ============ Content ============
      postText: {
        type: "string",
        description: "Text content of post",
      },
      postTextHash: {
        type: "string",
        description: "Hash of post text",
      },
      postedAt: {
        type: "string",
        description: "Post timestamp",
      },
      hasMedia: {
        type: "boolean",
        description: "Does post contain media?",
      },
      mediaType: {
        type: "string",
        enum: ["image", "video", "audio", "document", "link", "poll"],
        description: "Type of media attached",
      },
      mediaHash: {
        type: "string",
        description: "Hash of media content",
      },

      // ============ Engagement ============
      likeCount: {
        type: "number",
        description: "Likes/reactions at archive time",
      },
      shareCount: {
        type: "number",
        description: "Shares/retweets at archive time",
      },
      commentCount: {
        type: "number",
        description: "Comments at archive time",
      },

      // ============ Status ============
      isDeleted: {
        type: "boolean",
        description: "Has the post been deleted?",
      },
      isEdited: {
        type: "boolean",
        description: "Has the post been edited?",
      },
      editHistory: {
        type: "array",
        items: {
          type: "object",
          properties: {
            editedAt: { type: "string" },
            previousText: { type: "string" },
          },
        },
        description: "Edit history if available",
      },

      // ============ Context ============
      isReply: {
        type: "boolean",
        description: "Is this a reply to another post?",
      },
      replyToUrl: {
        type: "string",
        description: "URL of post being replied to",
      },
      threadPosition: {
        type: "number",
        description: "Position in thread (if part of thread)",
      },
    },
    required: [
      "platform",
      "postUrl",
      "archiveUrl",
      "accountHandle",
      "postText",
      "postedAt",
    ],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["archived_web"],
  minReferences: 1,

  active: true,
};

// ============ Exports ============

export const MEDIA_SCHEMAS = [
  NEWS_ARTICLE_SCHEMA,
  PRESS_RELEASE_SCHEMA,
  QUOTE_ATTRIBUTION_SCHEMA,
  SOCIAL_MEDIA_POST_SCHEMA,
];
