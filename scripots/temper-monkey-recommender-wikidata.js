// ==UserScript==
// @name         recommeender sematic web
// @namespace    http://tampermonkey.net/
// @version      0.1
// @description  try to take over the world!
// @author       Daniel Hoppe Alvarez
// @match        https://www.wikidata.org/wiki/*
// @grant    GM_addStyle
// @require http://code.jquery.com/jquery-3.3.1.min.js
// @include 5.189.164.102:8081/recommender
// @grant        GM_xmlhttpRequest
// ==/UserScript==

(function() {
    'use strict';

    // used endpoints
    const apiURL = "http://5.189.164.102:8081/recommender";
    const endpointUrl = 'https://query.wikidata.org/sparql'
    const wikidataURL = "http://www.wikidata.org/prop/direct/"
    const w3sparql  = "https://lov.linkeddata.es/dataset/lov/sparql"

    // whitelist
    const whitelist_url = "/prop/direct/"

    // class="wikibase-statementgroupview listview-item"
    var all_document_statements = document.querySelectorAll('.wikibase-statementgroupview,.listview-item');
    var amount_of_statements = all_document_statements.length;

    // create store for all statements
    var formatted_list_of_statements = [];
    var display_statements = [];

    for(var i = 0; i < amount_of_statements; i ++ ){
        // access a statement in list
        var cur_statement   = all_document_statements[i];
        var cur_property_id = cur_statement.id

        var link = wikidataURL + cur_property_id;
        formatted_list_of_statements.push(link)
    }

    GM_xmlhttpRequest ( {
      method:         "POST",
      url:            apiURL,
      responseType:   "json",
      data:           toStringArray(formatted_list_of_statements),
      onload:         processJSON_Response,
      onabort:        reportAJAX_Error,
      onerror:        reportAJAX_Error,
      ontimeout:      reportAJAX_Error
    });

   function processJSON_Response (rspObj) {

     // check if http request failed
     if(rspObj.status !== 200) {
         reportAJAX_Error(rspObj)
         return;
     }

     // go through all recommendations and get all
     // information needed to show them in front end
     var recommendations = rspObj.response;
       console.log(recommendations)
     for( var k = 0; k < recommendations.length; k++){

         // get recommendation
         var recommendation = recommendations[k];
         var recommendation_url = recommendations[k].Property.Str
         var propability = recommendation.Probability
         // check if is wikidata
         var isWikidata = recommendation_url.indexOf(whitelist_url) !== -1;

         if(isWikidata){
             console.log("\n", recommendation_url, "\n",recommendation.Probability)
             // get the infromation from the wikidata sparql endpoint
             makeSPARQLQuery(endpointUrl,propability, createQueryStr(recommendation_url), function( data, probability ) {

                 var bindings = data.results.bindings;
                 var info = {
                     propability: probability,
                     label: bindings[0].propLabel.value,
                     uri: bindings[0].prop.value
                 }

                 display_statements.push(info);
             })

         } /* else {

              makeSPARQLQuery(w3sparql, createQueryLovStr( recommendation_url), function( data ) {

                
                 var bindings = data.results.bindings;
                 if(bindings.length > 0){
                     var info = {
                         propability: recommendation.Probability,
                         label: bindings[0].propLabel.value,
                         uri: recommendation_url
                     }
                     console.log(info)
                     display_statements.push(info);
                 }

             })
             // console.log("other query ", recommendation_url)

         } */
     }
   }

console.log(display_statements)
    document.addEventListener('click', function (event) {
        // has to be the last step in the eventloop
        setTimeout(function(){
            // checks if list with propositions is existing
            var all_propositions = document.querySelectorAll('ul.ui-ooMenu.ui-widget.ui-widget-content.ui-suggester-list.ui-entityselector-list');
            // set max heigt of ul list
            $(all_propositions).css('max-height','250px');
            if (all_propositions.length === 0){
                return;
            }

            // replace all existing recommendations
            for (var i = 1; i < all_propositions.length; i++){

                $(all_propositions[i]).children().each(function(index) {

                    var item = display_statements[index]
                    var item_id = item.uri.replace(whitelist_url, "");

                    // override existing list recommendations with new ones
                    $(this).replaceWith(generateHTMLlistItem(item_id, item));
                })
            }

            all_propositions = sortByKey(all_propositions, 'Probability')
            // append all recommendations that are not replaced
            for (var j = all_propositions.length -1; j < display_statements.length; j++){
                var item = display_statements[j]
                var item_id = item.uri.replace(whitelist_url, "");
                $(all_propositions).append(generateHTMLlistItem(item_id, item))
            }

        }, 0);


    }, false);

   function createQueryStr(url){
    	return "SELECT ?prop ?propLabel {\n" +
        "  ?prop wikibase:directClaim <"+ url + "> .\n" +
        "   SERVICE wikibase:label {\n" +
        "     bd:serviceParam wikibase:language \"en\" .\n" +
        "   }\n" +
        "}\n" +
        "";
   }

   function createQueryLovStr(url){
       return "PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>\n"+
         "SELECT DISTINCT ?propLabel ?comment { "+
         "VALUES (?prop) {(<"+ url +">)} "+
         "GRAPH ?g { "+
         "?prop rdfs:label ?propLabel "+
         "OPTIONAL  { ?prop rdfs:comment ?comment } "+
         "}"+
        "}"
  }


    function sanitizeURL(url){
        var _url = url.replace("http://www.wikidata.org/prop/direct-normalized/", "http://www.wikidata.org/prop/direct/");
        return _url;
    }

   function makeSPARQLQuery( endpointUrl, probability, sparqlQuery, doneCallback ) {

        var settings = {
            headers: { Accept: 'application/sparql-results+json' },
            data: { query: sparqlQuery }
        };

        return $.ajax( endpointUrl, settings ).then( function (data) {doneCallback(data, probability)});
   }

   function reportAJAX_Error (rspObj) {
      console.error (`TM scrpt => Error ${rspObj.status}!  ${rspObj.statusText}`);
   }

   function generateHTMLlistItem(item_id, item){
      return '<li class="ui-ooMenu-item" dir="auto">' +
            '<a tabindex="-1" href="//www.wikidata.org/wiki/Property:' + item_id + '">' +
            '<span class="ui-entityselector-itemcontent">'+
            '<span class="ui-entityselector-label">'+ item.label +'</span>'+
            '<span class="ui-entityselector-description">' + 'The probability this entity should have this attribute is: '+ item.propability+'</span>' + '</span>' +
            '</a>' +
            '</li> '
    }

   // own toString method for an array as the built in method fails
   // hacky but works for now
   function toStringArray(array){
       var request = "[";

       for(var i = 0; i < array.length; i++){
           var toAdd = '"' + array[i] + '"';
           request += toAdd;

           if(i !== array.length - 1){
               request += ','
           }
       }

       request += "]";

       return request;
   }

   function sortByKey(array, key) {
    return array.sort(function(a, b) {
        var x = a[key]; var y = b[key];
        return ((x < y) ? -1 : ((x > y) ? 1 : 0));
    });
   }

})();