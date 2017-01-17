onmessage = function(e){
    let {data, colors} = e.data;

    let actual = data.map((m, i) => {
        // transform data to an array of [timestamp, value],
        // and sort by timestamp
        let data = Object.keys(m.dps)
            .sort((a,b) => (+a) - (+b))
            .map(ts => [ts*1000, m.dps[ts]])
            // HACK TODO - dont just work on the first 100 :>
            .slice(0,10000);

        return {
            metric: m.metric,
            color: colors[i][0],
            color2: colors[i][1],
            tags: m.tags,
            data: data
        };

    });
    postMessage(actual);
};
