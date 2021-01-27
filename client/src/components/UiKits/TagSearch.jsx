import React from 'react';
import Autocomplete from '@material-ui/lab/Autocomplete';
// import { makeStyles } from '@material-ui/core/styles';
import TextField from '@material-ui/core/TextField';

// const useStyles = makeStyles((theme) => ({
//     root: {
//         width: 500,
//         '& > * + *': {
//             marginTop: theme.spacing(3),
//         },
//     },
// }));

export default function Tags() {
    // const classes = useStyles();

    return (
        <Autocomplete
            multiple
            id="tags-outlined"
            options={top100Films}
            getOptionLabel={(option) => option.symbol}
            defaultValue={[top100Films[0]]}
            filterSelectedOptions
            renderInput={(params) => (
                <TextField {...params} variant="outlined" label="Search Tickers" placeholder="Add" />
            )}
        />
    );
}

const top100Films = [
    { symbol: 'spy' },
    { symbol: 'spxl' },
    { symbol: '^skew' },
    { symbol: 'tlt' },
    { symbol: 'gld' },
    { symbol: 'gldm' },
];
