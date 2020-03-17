( function ( mw, ps ) {
	'use strict';

	mw.hook( 'wikibase.entityselector.search' ).add( function ( data, addPromise ) {

		var suggester = new ps.PropertySuggester( data.element );
		if ( !suggester.useSuggester( data.options.type ) ) {
			return;
		}

		addPromise(
			suggester.getSuggestions( data.options.url, data.options.language, data.term )
		);

	} );

}( mediaWiki, propertySuggester ) );
