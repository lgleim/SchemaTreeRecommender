<?php

namespace PropertySuggester;

use ApiResult;
use PropertySuggester\Suggesters\Suggestion;
use Wikibase\TermIndexEntry;
use Wikibase\TermIndex;
use Wikibase\DataModel\Entity\EntityId;
use Wikibase\Lib\Store\EntityTitleLookup;

/**
 * ResultBuilder builds Json-compatible array structure from suggestions
 *
 * @author BP2013N2
 * @license GPL-2.0-or-later
 */
class ResultBuilder {

	/**
	 * @var EntityTitleLookup
	 */
	private $entityTitleLookup;

	/**
	 * @var TermIndex
	 */
	private $termIndex;

	/**
	 * @var ApiResult
	 */
	private $result;

	/**
	 * @var string
	 */
	private $searchPattern;

	/**
	 * @param ApiResult $result
	 * @param TermIndex $termIndex
	 * @param EntityTitleLookup $entityTitleLookup
	 * @param string $search
	 */
	public function __construct(
		ApiResult $result,
		TermIndex $termIndex,
		EntityTitleLookup $entityTitleLookup,
		$search
	) {
		$this->entityTitleLookup = $entityTitleLookup;
		$this->termIndex = $termIndex;
		$this->result = $result;
		$this->searchPattern = '/^' . preg_quote( $search, '/' ) . '/i';
	}

	/**
	 * @param Suggestion[] $suggestions
	 * @param string $language
	 * @return array[]
	 */
	public function createResultArray( array $suggestions, $language ) {
		$entries = [];
		$ids = [];
		foreach ( $suggestions as $suggestion ) {
			$id = $suggestion->getPropertyId();
			$ids[] = $id;
		}
		// See SearchEntities
		$terms = $this->termIndex->getTermsOfEntities(
			$ids,
			null,
			[ $language ]
		);
		$clusteredTerms = $this->clusterTerms( $terms );

		foreach ( $suggestions as $suggestion ) {
			$id = $suggestion->getPropertyId();
			$entries[] = $this->buildEntry( $id, $clusteredTerms, $suggestion );
		}
		return $entries;
	}

	/**
	 * @param EntityId $id
	 * @param array[] $clusteredTerms
	 * @param Suggestion $suggestion
	 * @return array
	 */
	private function buildEntry( EntityId $id, array $clusteredTerms, Suggestion $suggestion ) {
		$entry = [
			'id' => $id->getSerialization(),
			'url' => $this->entityTitleLookup->getTitleForId( $id )->getFullURL(),
			'rating' => $suggestion->getProbability(),
		];

		/** @var TermIndexEntry[] $matchingTerms */
		if ( isset( $clusteredTerms[$id->getSerialization()] ) ) {
			$matchingTerms = $clusteredTerms[$id->getSerialization()];
		} else {
			$matchingTerms = [];
		}

		foreach ( $matchingTerms as $term ) {
			switch ( $term->getTermType() ) {
				case TermIndexEntry::TYPE_LABEL:
					$entry['label'] = $term->getText();
					break;
				case TermIndexEntry::TYPE_DESCRIPTION:
					$entry['description'] = $term->getText();
					break;
				case TermIndexEntry::TYPE_ALIAS:
					$this->checkAndSetAlias( $entry, $term );
					break;
			}
		}

		if ( !isset( $entry['label'] ) ) {
			$entry['label'] = $id->getSerialization();
		} elseif ( preg_match( $this->searchPattern, $entry['label'] ) ) {
			// No aliases needed in the output when the label already is a successful match.
			unset( $entry['aliases'] );
		}

		return $entry;
	}

	/**
	 * @param TermIndexEntry[] $terms
	 * @return TermIndexEntry[][]
	 */
	private function clusterTerms( array $terms ) {
		$clusteredTerms = [];

		foreach ( $terms as $term ) {
			$id = $term->getEntityId()->getSerialization();
			if ( !isset( $clusteredTerms[$id] ) ) {
				$clusteredTerms[$id] = [];
			}
			$clusteredTerms[$id][] = $term;
		}
		return $clusteredTerms;
	}

	/**
	 * @param array $entry
	 * @param TermIndexEntry $term
	 */
	private function checkAndSetAlias( array &$entry, TermIndexEntry $term ) {
		// Do not add more than one matching alias to the "aliases" field.
		if ( !empty( $entry['aliases'] ) ) {
			return;
		}

		if ( preg_match( $this->searchPattern, $term->getText() ) ) {
			if ( !isset( $entry['aliases'] ) ) {
				$entry['aliases'] = [];
				ApiResult::setIndexedTagName( $entry['aliases'], 'alias' );
			}
			$entry['aliases'][] = $term->getText();
		}
	}

	/**
	 * @param array[] $entries
	 * @param array[] $searchResults
	 * @param int $resultSize
	 * @return array[] representing Json
	 */
	public function mergeWithTraditionalSearchResults( array $entries, array $searchResults, $resultSize ) {
		// Avoid duplicates
		$existingKeys = [];
		foreach ( $entries as $entry ) {
			$existingKeys[$entry['id']] = true;
		}

		$distinctCount = count( $entries );
		foreach ( $searchResults as $result ) {
			if ( !array_key_exists( $result['id'], $existingKeys ) ) {
				$entries[] = $result;
				$distinctCount++;
				if ( $distinctCount >= $resultSize ) {
					break;
				}
			}
		}
		return $entries;
	}

}
