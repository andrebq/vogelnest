<script>
    import endpoints from "./endpoints";
    import { List, OrderedSet } from 'immutable';
import { update_keyed_each } from "svelte/internal";

    export let tweetCount = 0;
    export let streamConnected = false;
    export let maxSize = 20;
    export let latestTweets = List([]);
    export let hashTags = OrderedSet([]);
    export let mentions = OrderedSet([]);

    console.info('ws endpoint', endpoints.websocket());
    let ws = new WebSocket(endpoints.websocket());
    ws.onopen = function()  {
        console.info('connected');
        streamConnected = true;
    };
    ws.onmessage = function(msg) {
        tweetCount++;
        const tweet = JSON.parse(msg.data);
        hashTags = hashTags.withMutations((set) => {
            tweet.entities.hashtags.forEach((ht) => set.add(ht.text));
            return set;
        });
        mentions = mentions.withMutations((set) => {
            tweet.entities.user_mentions.forEach((ut) => set.add(ut.screen_name))
            return set;
        })
        latestTweets = trim(latestTweets.unshift(tweet));
    }

    function trim(lst) {
        if (lst.size > maxSize) {
            lst = lst.delete(lst.size-1);
        }
        return lst;
    }
</script>

<section>
    <article>
        <h1>
            Tweet stream.... Oh my!
        </h1>

        <main>
        {#if streamConnected}
            <p><strong>{tweetCount}</strong> tweets so far!</p>

            <ul>
                {#each hashTags.toJS() as ht}
                    <span>#{ht} </span>
                {/each}
            </ul>

            <ul>
            {#each latestTweets.toJS() as tweet}
                {#if tweet.truncated }
                    <li>{tweet.extended_tweet.full_text}</li>
                {:else}
                    <li>{tweet.text}</li>
                {/if}
            {/each}
            </ul>

            <ul>
                {#each mentions.toJS() as mention}
                    <li>{mention}</li>
                {/each}
            </ul>

        {:else}
            <p>Stream not connected...</p>
        {/if}
        </main>
    </article>
</section>
