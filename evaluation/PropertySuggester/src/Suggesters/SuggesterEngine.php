<?php

namespace PropertySuggester\Suggesters;

use Wikibase\DataModel\Entity\Item;
use Wikibase\DataModel\Entity\PropertyId;

/**
 * interface for (Property-)Suggester
 *
 * @author BP2013N2
 * @license GPL-2.0-or-later
 */
interface SuggesterEngine {

	/**
	 * Suggest only properties that might be added (non-deprecated, not yet present)
	 */
	const SUGGEST_NEW = 'new';

	/**
	 * Suggest everything, even already present or deprecated values.
	 */
	const SUGGEST_ALL = 'all';

	/**
	 * Returns suggested attributes
	 *
	 * @param PropertyId[] $propertyIds
	 * @param int $limit
	 * @param float $minProbability
	 * @param string $context
	 * @param string $include One of the self::SUGGEST_* constants
	 * @return Suggestion[]
	 */
	public function suggestByPropertyIds( array $propertyIds, $limit, $minProbability, $context, $include );

	/**
	 * Returns suggested attributes
	 *
	 * @param Item $item
	 * @param int $limit
	 * @param float $minProbability
	 * @param string $context
	 * @param string $include One of the self::SUGGEST_* constants
	 * @return Suggestion[]
	 */
	public function suggestByItem( Item $item, $limit, $minProbability, $context, $include );

}
