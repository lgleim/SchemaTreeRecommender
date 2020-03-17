window.propertySuggester = window.propertySuggester || {};

window.propertySuggester.PropertySuggester = ( function ( $ ) {
	'use strict';

	function SELF( $element ) {
		this._$element = $element;
	}

	/**
	 * @property {string}
	 * @private
	 */
	SELF.prototype._$element = null;

	/**
	 * @public
	 * @param {string} type entity type
	 * @return {boolean}
	 */
	SELF.prototype.useSuggester = function ( type ) {
		var entity = this._getEntity();

		return type === 'property' &&
			entity && entity.getType() === 'item' &&
			this._getPropertyContext() !== null;
	};

	/**
	 * Get the entity from the surrounding entityview or return null
	 *
	 * @private
	 * @return {wikibase.Entity|null}
	 */
	SELF.prototype._getEntity = function () {
		var $entityView;

		try {
			$entityView = this._$element.closest( ':wikibase-entityview' );
		} catch ( ex ) {
			return null;
		}

		if ( $entityView.length > 0 ) {
			return $entityView.data( 'entityview' ).option( 'value' );
		}

		return null;
	};

	/**
	 * Returns the property id for the enclosing statementview or null if no property is
	 * selected yet.
	 *
	 * @private
	 * @return {string|null}
	 */
	SELF.prototype._getPropertyId = function () {
		var $statementview,
			statement;

		try {
			$statementview = this._$element.closest( ':wikibase-statementview' );
		} catch ( ex ) {
			return null;
		}

		if ( $statementview.length > 0 ) {
			statement = $statementview.data( 'statementview' ).option( 'value' );
			if ( statement ) {
				return statement.getClaim().getMainSnak().getPropertyId();
			}
		}

		return null;
	};

	/**
	 * Returns either 'item', 'qualifier', 'reference' or null depending on the context of the
	 * entityselector. 'item' is returned in case that the selector is for a new property in an
	 * item.
	 *
	 * @private
	 * @return {string|null}
	 */
	SELF.prototype._getPropertyContext = function () {
		if ( this._isInNewStatementView() ) {
			if ( !this._isQualifier() && !this._isReference() ) {
				return 'item';
			}
		} else if ( this._isQualifier() ) {
			return 'qualifier';
		} else if ( this._isReference() ) {
			return 'reference';
		}

		return null;
	};

	/**
	 * @private
	 * @return {boolean}
	 */
	SELF.prototype._isQualifier = function () {
		var $statementview = this._$element.closest( ':wikibase-statementview' ),
			statementview = $statementview.data( 'statementview' );

		if ( !statementview ) {
			return false;
		}

		return this._$element.closest( statementview.$qualifiers ).length > 0;
	};

	/**
	 * @private
	 * @return {boolean}
	 */
	SELF.prototype._isReference = function () {
		var $referenceview = this._$element.closest( ':wikibase-referenceview' );

		return $referenceview.length > 0;
	};

	/**
	 * detect if this is a new statement view.
	 *
	 * @private
	 * @return {boolean}
	 */
	SELF.prototype._isInNewStatementView = function () {
		var $statementview = this._$element.closest( ':wikibase-statementview' );

		if ( $statementview.length > 0 ) {
			return !$statementview.data( 'statementview' ).option( 'value' );
		}

		return true;
	};

	/**
	 * Get suggestions
	 *
	 * @public
	 * @param {string} url of api endpoint
	 * @param {string} language
	 * @param {term} term search term
	 * @return {jQuery.Promise}
	 */
	SELF.prototype.getSuggestions = function ( url, language, term ) {
		var $deferred = $.Deferred(),
			data = {
				action: 'wbsgetsuggestions',
				search: term,
				context: this._getPropertyContext(),
				format: 'json',
				language: language
			};

		if ( data.context === 'item' ) {
			data.entity = this._getEntity().getId();
		} else {
			data.properties = this._getPropertyId();
		}

		return $.getJSON( url, data ).then( function ( d ) {
			if ( !d.search ) {
				return $deferred.resolve().promise();
			}

			return $deferred.resolve( d.search ).promise();
		} );

	};

	return SELF;
}( jQuery ) );
