<?php

namespace PropertySuggester;

use InvalidArgumentException;
use PropertySuggester\Suggesters\SuggesterEngine;
use PropertySuggester\Suggesters\Suggestion;
use Wikibase\DataModel\Entity\Item;
use Wikibase\DataModel\Entity\ItemId;
use Wikibase\DataModel\Entity\PropertyId;
use Wikibase\DataModel\Services\Lookup\EntityLookup;
use Wikibase\Repo\Api\EntitySearchHelper;

/**
 * API module helper to generate property suggestions.
 *
 * @author BP2013N2
 * @license GPL-2.0-or-later
 */
class SuggestionGenerator {

	/**
	 * @var EntityLookup
	 */
	private $entityLookup;

	/**
	 * @var EntitySearchHelper
	 */
	private $entityTermSearchHelper;

	/**
	 * @var SuggesterEngine
	 */
	private $suggester;

	public function __construct(
		EntityLookup $entityLookup,
		EntitySearchHelper $entityTermSearchHelper,
		SuggesterEngine $suggester
	) {
		$this->entityLookup = $entityLookup;
		$this->entityTermSearchHelper = $entityTermSearchHelper;
		$this->suggester = $suggester;
	}

	/**
	 * @param string $itemIdString
	 * @param int $limit
	 * @param float $minProbability
	 * @param string $context
	 * @param string $include One of the SuggesterEngine::SUGGEST_* constants
	 * @throws InvalidArgumentException
	 * @return Suggestion[]
	 */
	public function generateSuggestionsByItem( $itemIdString, $limit, $minProbability, $context, $include ) {
		$itemId = new ItemId( $itemIdString );
		/** @var Item $item */
		$item = $this->entityLookup->getEntity( $itemId );

		if ( $item === null ) {
			throw new InvalidArgumentException( 'Item ' . $itemIdString . ' could not be found' );
		}

		return $this->suggester->suggestByItem( $item, $limit, $minProbability, $context, $include );
	}

	/**
	 * @param string[] $propertyIdList - A list of property-id-strings
	 * @param int $limit
	 * @param float $minProbability
	 * @param string $context
	 * @param string $include One of the SuggesterEngine::SUGGEST_* constants
	 * @return Suggestion[]
	 */
	public function generateSuggestionsByPropertyList(
		array $propertyIdList,
		$limit,
		$minProbability,
		$context,
		$include
	) {
		$propertyIds = [];
		foreach ( $propertyIdList as $stringId ) {
			$propertyIds[] = $stringId[0]=="P" ? new PropertyId( $stringId ) : new ItemId ($stringId);
		}

		$suggestions = $this->suggester->suggestByPropertyIds(
			$propertyIds,
			$limit,
			$minProbability,
			$context,
			$include
		);

		return $suggestions;
	}

	/**
	 * @param Suggestion[] $suggestions
	 * @param string $search
	 * @param string $language
	 * @param int $resultSize
	 * @return Suggestion[]
	 */
	public function filterSuggestions( array $suggestions, $search, $language, $resultSize ) {
		if ( !$search ) {
			return array_slice( $suggestions, 0, $resultSize );
		}

		$searchResults = $this->entityTermSearchHelper->getRankedSearchResults(
			$search,
			$language,
			'property',
			$resultSize,
			true
		);

		$id_set = [];
		foreach ( $searchResults as $searchResult ) {
			$id_set[$searchResult->getEntityId()->getNumericId()] = true;
		}

		$matching_suggestions = [];
		$count = 0;
		foreach ( $suggestions as $suggestion ) {
			if ( array_key_exists( $suggestion->getPropertyId()->getNumericId(), $id_set ) ) {
				$matching_suggestions[] = $suggestion;
				if ( ++$count === $resultSize ) {
					break;
				}
			}
		}
		return $matching_suggestions;
	}

}
