pushd charts
for chart in *; do
  echo $chart
  if [ -d "$chart" ]; then
    pushd "$chart"
    name=$(yq .name Chart.yaml)
    "$HOME"/orch-ci/scripts/version-tag-param.sh "${name}/v"
    popd
  fi
done
popd