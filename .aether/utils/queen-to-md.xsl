<?xml version="1.0" encoding="UTF-8"?>
<!--
  XSLT Stylesheet: queen-wisdom to Markdown

  Purpose: Transform queen-wisdom.xml into human-readable QUEEN.md format
           for display and documentation purposes.

  Usage: xsltproc queen-to-md.xsl queen-wisdom.xml > QUEEN.md

  Version: 1.0.0
  Namespace: http://aether.colony/schemas/queen-wisdom/1.0
-->
<xsl:stylesheet version="1.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <xsl:output method="text" encoding="UTF-8" indent="no"/>

  <!-- ============================================================ -->
  <!-- Main Template                                                -->
  <!-- ============================================================ -->

  <xsl:template match="/queen-wisdom">
    <xsl:text># QUEEN.md — Colony Wisdom&#10;&#10;</xsl:text>

    <!-- Metadata Header -->
    <xsl:text>&gt; Last evolved: </xsl:text>
    <xsl:value-of select="metadata/modified"/>
    <xsl:text>&#10;</xsl:text>

    <xsl:text>&gt; Colonies contributed: </xsl:text>
    <xsl:value-of select="count(//evidence/colony-ref/@id)"/>
    <xsl:text>&#10;</xsl:text>

    <xsl:text>&gt; Wisdom version: </xsl:text>
    <xsl:value-of select="metadata/version"/>
    <xsl:text>&#10;&#10;</xsl:text>

    <xsl:text>---&#10;&#10;</xsl:text>

    <!-- Philosophies Section -->
    <xsl:apply-templates select="philosophies"/>

    <!-- Patterns Section -->
    <xsl:apply-templates select="patterns"/>

    <!-- Redirects Section -->
    <xsl:apply-templates select="redirects"/>

    <!-- Stack Wisdom Section -->
    <xsl:apply-templates select="stack-wisdom"/>

    <!-- Decrees Section -->
    <xsl:apply-templates select="decrees"/>

    <!-- Evolution Log Section -->
    <xsl:apply-templates select="evolution-log"/>

    <xsl:text>---&#10;&#10;</xsl:text>

    <!-- JSON Metadata Block (for backward compatibility) -->
    <xsl:call-template name="metadata-block"/>
  </xsl:template>

  <!-- ============================================================ -->
  <!-- Section Templates                                            -->
  <!-- ============================================================ -->

  <xsl:template match="philosophies">
    <xsl:text>## 📜 Philosophies&#10;&#10;</xsl:text>
    <xsl:text>Core beliefs that guide all colony work. These are validated through repeated successful application across multiple colonies.&#10;&#10;</xsl:text>

    <xsl:for-each select="philosophy">
      <xsl:text>- **</xsl:text>
      <xsl:value-of select="@id"/>
      <xsl:text>** (</xsl:text>
      <xsl:value-of select="@created_at"/>
      <xsl:text>): </xsl:text>
      <xsl:value-of select="content"/>
      <xsl:text>&#10;</xsl:text>

      <!-- Principles subsection if present -->
      <xsl:if test="principles/principle">
        <xsl:text>  - Principles:&#10;</xsl:text>
        <xsl:for-each select="principles/principle">
          <xsl:text>    - </xsl:text>
          <xsl:value-of select="."/>
          <xsl:text>&#10;</xsl:text>
        </xsl:for-each>
      </xsl:if>

      <!-- Evidence subsection -->
      <xsl:if test="evidence/colony-ref">
        <xsl:text>  - Validated by: </xsl:text>
        <xsl:for-each select="evidence/colony-ref">
          <xsl:value-of select="@id"/>
          <xsl:if test="position() != last()">, </xsl:if>
        </xsl:for-each>
        <xsl:text>&#10;</xsl:text>
      </xsl:if>
    </xsl:for-each>

    <xsl:text>&#10;</xsl:text>
  </xsl:template>

  <xsl:template match="patterns">
    <xsl:text>## 🧭 Patterns&#10;&#10;</xsl:text>
    <xsl:text>Validated approaches that consistently work. These represent discovered best practices that have proven themselves in the field.&#10;&#10;</xsl:text>

    <xsl:for-each select="pattern">
      <xsl:text>- **</xsl:text>
      <xsl:value-of select="@id"/>
      <xsl:text>**</xsl:text>

      <xsl:if test="pattern_type">
        <xsl:text> [</xsl:text>
        <xsl:value-of select="pattern_type"/>
        <xsl:text>]</xsl:text>
      </xsl:if>

      <xsl:text> (</xsl:text>
      <xsl:value-of select="@created_at"/>
      <xsl:text>): </xsl:text>
      <xsl:value-of select="content"/>
      <xsl:text>&#10;</xsl:text>

      <!-- Detection criteria if present -->
      <xsl:if test="detection_criteria">
        <xsl:text>  - When to apply: </xsl:text>
        <xsl:value-of select="detection_criteria"/>
        <xsl:text>&#10;</xsl:text>
      </xsl:if>

      <!-- Examples if present -->
      <xsl:if test="examples/example">
        <xsl:text>  - Examples:&#10;</xsl:text>
        <xsl:for-each select="examples/example">
          <xsl:text>    - Scenario: </xsl:text>
          <xsl:value-of select="scenario"/>
          <xsl:text> → Application: </xsl:text>
          <xsl:value-of select="application"/>
          <xsl:if test="outcome">
            <xsl:text> (Outcome: </xsl:text>
            <xsl:value-of select="outcome"/>
            <xsl:text>)</xsl:text>
          </xsl:if>
          <xsl:text>&#10;</xsl:text>
        </xsl:for-each>
      </xsl:if>
    </xsl:for-each>

    <xsl:text>&#10;</xsl:text>
  </xsl:template>

  <xsl:template match="redirects">
    <xsl:text>## ⚠️ Redirects&#10;&#10;</xsl:text>
    <xsl:text>Anti-patterns to avoid. These represent approaches that have caused problems and should be redirected away from.&#10;&#10;</xsl:text>

    <xsl:for-each select="redirect">
      <xsl:text>- **</xsl:text>
      <xsl:value-of select="@id"/>
      <xsl:text>**</xsl:text>

      <xsl:if test="@confidence">
        <xsl:text> [strength: </xsl:text>
        <xsl:value-of select="@confidence"/>
        <xsl:text>]</xsl:text>
      </xsl:if>

      <xsl:text> (</xsl:text>
      <xsl:value-of select="@created_at"/>
      <xsl:text>): </xsl:text>
      <xsl:value-of select="content"/>
      <xsl:text>&#10;</xsl:text>

      <!-- Constraint type if present -->
      <xsl:if test="constraint_type">
        <xsl:text>  - Constraint: </xsl:text>
        <xsl:value-of select="constraint_type"/>
        <xsl:text>&#10;</xsl:text>
      </xsl:if>

      <!-- Context if present -->
      <xsl:if test="context">
        <xsl:text>  - Context: </xsl:text>
        <xsl:value-of select="context"/>
        <xsl:text>&#10;</xsl:text>
      </xsl:if>
    </xsl:for-each>

    <xsl:text>&#10;</xsl:text>
  </xsl:template>

  <xsl:template match="stack-wisdom">
    <xsl:text>## 🔧 Stack Wisdom&#10;&#10;</xsl:text>
    <xsl:text>Technology-specific insights and constraints detected through codebase analysis.&#10;&#10;</xsl:text>

    <xsl:for-each select="wisdom">
      <xsl:text>- **</xsl:text>
      <xsl:value-of select="@id"/>
      <xsl:text>**</xsl:text>

      <xsl:if test="technology">
        <xsl:text> [</xsl:text>
        <xsl:value-of select="technology"/>
        <xsl:text>]</xsl:text>
      </xsl:if>

      <xsl:text> (</xsl:text>
      <xsl:value-of select="@created_at"/>
      <xsl:text>): </xsl:text>
      <xsl:value-of select="content"/>
      <xsl:text>&#10;</xsl:text>

      <!-- Version range if present -->
      <xsl:if test="version_range">
        <xsl:text>  - Applies to: </xsl:text>
        <xsl:value-of select="version_range"/>
        <xsl:text>&#10;</xsl:text>
      </xsl:if>

      <!-- Workaround if present -->
      <xsl:if test="workaround">
        <xsl:text>  - Workaround: </xsl:text>
        <xsl:value-of select="workaround"/>
        <xsl:text>&#10;</xsl:text>
      </xsl:if>
    </xsl:for-each>

    <xsl:text>&#10;</xsl:text>
  </xsl:template>

  <xsl:template match="decrees">
    <xsl:text>## 🏛️ Decrees&#10;&#10;</xsl:text>
    <xsl:text>User-mandated rules that override other guidance. These represent explicit directives from the Queen.&#10;&#10;</xsl:text>

    <xsl:for-each select="decree">
      <xsl:text>- **</xsl:text>
      <xsl:value-of select="@id"/>
      <xsl:text>**</xsl:text>

      <xsl:if test="scope">
        <xsl:text> [</xsl:text>
        <xsl:value-of select="scope"/>
        <xsl:text>]</xsl:text>
      </xsl:if>

      <xsl:text> (</xsl:text>
      <xsl:value-of select="@created_at"/>
      <xsl:text>): </xsl:text>
      <xsl:value-of select="content"/>
      <xsl:text>&#10;</xsl:text>

      <!-- Authority if present -->
      <xsl:if test="authority">
        <xsl:text>  - Authority: </xsl:text>
        <xsl:value-of select="authority"/>
        <xsl:text>&#10;</xsl:text>
      </xsl:if>

      <!-- Expiration if present -->
      <xsl:if test="expiration">
        <xsl:text>  - Expires: </xsl:text>
        <xsl:value-of select="expiration"/>
        <xsl:text>&#10;</xsl:text>
      </xsl:if>
    </xsl:for-each>

    <xsl:text>&#10;</xsl:text>
  </xsl:template>

  <xsl:template match="evolution-log">
    <xsl:text>## 📊 Evolution Log&#10;&#10;</xsl:text>
    <xsl:text>Track how wisdom has evolved over time.&#10;&#10;</xsl:text>

    <xsl:text>| Date | Colony | Change | Details |&#10;</xsl:text>
    <xsl:text>|------|--------|--------|---------|&#10;</xsl:text>

    <xsl:for-each select="entry">
      <xsl:text>| </xsl:text>
      <xsl:value-of select="@timestamp"/>
      <xsl:text> | </xsl:text>
      <xsl:value-of select="@colony"/>
      <xsl:text> | </xsl:text>
      <xsl:value-of select="@action"/>
      <xsl:text> | </xsl:text>

      <!-- Build details string -->
      <xsl:choose>
        <xsl:when test="@type and @from">
          <xsl:text>Promoted </xsl:text>
          <xsl:value-of select="@type"/>
          <xsl:text> from </xsl:text>
          <xsl:value-of select="@from"/>
        </xsl:when>
        <xsl:when test="@type">
          <xsl:text>Added: </xsl:text>
          <xsl:value-of select="@type"/>
        </xsl:when>
        <xsl:otherwise>
          <xsl:text>Entry recorded</xsl:text>
        </xsl:otherwise>
      </xsl:choose>

      <xsl:if test="note">
        <xsl:text> - </xsl:text>
        <xsl:value-of select="note"/>
      </xsl:if>

      <xsl:text> |&#10;</xsl:text>
    </xsl:for-each>

    <xsl:text>&#10;</xsl:text>
  </xsl:template>

  <!-- ============================================================ -->
  <!-- Metadata Block Template                                      -->
  <!-- ============================================================ -->

  <xsl:template name="metadata-block">
    <xsl:text>&lt;!-- METADATA&#10;</xsl:text>
    <xsl:text>{&#10;</xsl:text>

    <xsl:text>  "version": "</xsl:text>
    <xsl:value-of select="metadata/version"/>
    <xsl:text>",&#10;</xsl:text>

    <xsl:text>  "last_evolved": "</xsl:text>
    <xsl:value-of select="metadata/modified"/>
    <xsl:text>",&#10;</xsl:text>

    <!-- Count colonies contributed -->
    <xsl:text>  "colonies_contributed": [</xsl:text>
    <xsl:for-each select="//evidence/colony-ref[not(@id=preceding::colony-ref/@id)]">
      <xsl:text>"</xsl:text>
      <xsl:value-of select="@id"/>
      <xsl:text>"</xsl:text>
      <xsl:if test="position() != last()">, </xsl:if>
    </xsl:for-each>
    <xsl:text>],&#10;</xsl:text>

    <!-- Promotion thresholds -->
    <xsl:text>  "promotion_thresholds": {&#10;</xsl:text>
    <xsl:text>    "philosophy": 5,&#10;</xsl:text>
    <xsl:text>    "pattern": 3,&#10;</xsl:text>
    <xsl:text>    "redirect": 2,&#10;</xsl:text>
    <xsl:text>    "stack": 1,&#10;</xsl:text>
    <xsl:text>    "decree": 0&#10;</xsl:text>
    <xsl:text>  },&#10;</xsl:text>

    <!-- Stats -->
    <xsl:text>  "stats": {&#10;</xsl:text>
    <xsl:text>    "total_philosophies": </xsl:text>
    <xsl:value-of select="count(//philosophy)"/>
    <xsl:text>,&#10;</xsl:text>

    <xsl:text>    "total_patterns": </xsl:text>
    <xsl:value-of select="count(//pattern)"/>
    <xsl:text>,&#10;</xsl:text>

    <xsl:text>    "total_redirects": </xsl:text>
    <xsl:value-of select="count(//redirect)"/>
    <xsl:text>,&#10;</xsl:text>

    <xsl:text>    "total_stack_entries": </xsl:text>
    <xsl:value-of select="count(//stack-wisdom/wisdom)"/>
    <xsl:text>,&#10;</xsl:text>

    <xsl:text>    "total_decrees": </xsl:text>
    <xsl:value-of select="count(//decree)"/>
    <xsl:text>&#10;</xsl:text>

    <xsl:text>  }&#10;</xsl:text>
    <xsl:text>}&#10;</xsl:text>
    <xsl:text>--&gt;&#10;</xsl:text>
  </xsl:template>

  <!-- ============================================================ -->
  <!-- Utility Templates                                            -->
  <!-- ============================================================ -->

  <!-- Strip trailing/leading whitespace -->
  <xsl:template name="trim">
    <xsl:param name="string"/>
    <xsl:value-of select="normalize-space($string)"/>
  </xsl:template>

  <!-- Escape markdown special characters in content -->
  <xsl:template name="escape-markdown">
    <xsl:param name="text"/>
    <!-- Note: In XSLT 1.0, complex string replacement requires recursive templates -->
    <!-- For simplicity, we assume content doesn't contain problematic markdown -->
    <xsl:value-of select="$text"/>
  </xsl:template>

</xsl:stylesheet>
