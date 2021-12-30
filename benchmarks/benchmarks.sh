curl https://s3.amazonaws.com/nyc-tlc/trip+data/yellow_tripdata_2021-04.csv -o taxi.csv

echo "CREATE EXTERNAL TABLE taxi
STORED AS CSV
WITH HEADER ROW
LOCATION './taxi.csv';

SELECT passenger_count, COUNT(*), AVG(total_amount) FROM taxi GROUP BY passenger_count" > datafusion_commands.txt

wc -l taxi.csv && wc -l taxi.csv && wc -l taxi.csv && wc -l taxi.csv # Get the cache warm.

hyperfine --min-runs 10 -w 2 --export-markdown benchmarks.md \
'OCTOSQL_NO_TELEMETRY=1 octosql "SELECT passenger_count, COUNT(*), AVG(total_amount) FROM taxi.csv GROUP BY passenger_count"' \
"q -d ',' -H \"SELECT passenger_count, COUNT(*), AVG(total_amount) FROM taxi.csv GROUP BY passenger_count\"" \
"q -d ',' -H -C readwrite \"SELECT passenger_count, COUNT(*), AVG(total_amount) FROM taxi.csv GROUP BY passenger_count\"" \
'textql -header -sql "SELECT passenger_count, COUNT(*), AVG(total_amount) FROM taxi GROUP BY passenger_count" taxi.csv' \
'datafusion-cli -f datafusion_commands.txt'

