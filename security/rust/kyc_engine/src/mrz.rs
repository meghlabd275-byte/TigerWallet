//! ICAO 9303 machine-readable zone parsing and validation.
//!
//! Every check digit defined by ICAO Doc 9303 part 3 is computed and verified
//! here, including the composite check digits, which are the part of the
//! document that actually catches a mistyped or forged number. A document that
//! fails a check digit is reported as invalid; it is never silently accepted
//! with the bad field blanked out.
//!
//! Only TD3 (passport booklet) and TD1 (identity card) formats are parsed,
//! because those are the two document types this platform accepts.

use crate::errors::KycError;

/// Maximum accepted length of a machine-readable zone, from ICAO 9303.
/// TD3 is 2 lines of 44 characters; TD1 is 3 lines of 30.
const TD3_LINE_LEN: usize = 44;
const TD1_LINE_LEN: usize = 30;

/// The 7-3-1 repeating weight used by every ICAO 9303 check digit.
const CHECK_WEIGHTS: [u32; 3] = [7, 3, 1];

/// Document format detected from the MRZ.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum MrzFormat {
    /// Passport booklet: 2 lines of 44 characters.
    Td3,
    /// Identity card: 3 lines of 30 characters.
    Td1,
}

/// Sex as encoded in the MRZ.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Sex {
    Male,
    Female,
    Unspecified,
}

/// A parsed and fully check-digit-verified machine-readable zone.
///
/// A value of this type existing at all means every ICAO 9303 check digit
/// passed. Callers can rely on that and do not need to re-validate.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Mrz {
    pub format: MrzFormat,
    pub document_type: String,
    pub issuing_country: String,
    pub document_number: String,
    pub nationality: String,
    pub surname: String,
    pub given_names: String,
    pub date_of_birth: String,
    pub sex: Sex,
    pub date_of_expiry: String,
    pub personal_number: Option<String>,
}

impl Mrz {
    /// The full name as it should be compared against a self-declaration,
    /// normalised to upper case with single spaces.
    pub fn normalised_name(&self) -> String {
        let mut parts: Vec<&str> = Vec::new();
        if !self.given_names.is_empty() {
            parts.push(&self.given_names);
        }
        if !self.surname.is_empty() {
            parts.push(&self.surname);
        }
        parts.join(" ").to_uppercase()
    }
}

/// Converts one MRZ character to its check-digit value.
///
/// Digits map to themselves, letters map to 10..=35, and the filler `<` is 0.
/// Anything else is rejected rather than coerced, because coercing an
/// unexpected character would let a malformed document produce a valid-looking
/// check digit.
fn char_value(c: u8) -> Result<u32, KycError> {
    match c {
        b'0'..=b'9' => Ok(u32::from(c - b'0')),
        b'A'..=b'Z' => Ok(u32::from(c - b'A') + 10),
        b'<' => Ok(0),
        _ => Err(KycError::MrzInvalidCharacter { character: c as char }),
    }
}

/// Computes an ICAO 9303 check digit over `input`.
///
/// The check digit is the sum of each character's value weighted by the
/// repeating 7-3-1 pattern, modulo 10.
pub fn check_digit(input: &str) -> Result<u8, KycError> {
    let mut sum: u32 = 0;
    for (i, b) in input.bytes().enumerate() {
        let v = char_value(b)?;
        sum += v * CHECK_WEIGHTS[i % 3];
    }
    Ok((sum % 10) as u8)
}

/// Verifies a check digit character against the value computed over `input`.
pub fn verify_check_digit(input: &str, expected: u8) -> Result<(), KycError> {
    if !expected.is_ascii_digit() {
        return Err(KycError::MrzCheckDigitNotNumeric {
            field: input.to_string(),
            found: expected as char,
        });
    }
    let computed = check_digit(input)?;
    let found = expected - b'0';
    if computed != found {
        return Err(KycError::MrzCheckDigitMismatch {
            field: input.to_string(),
            expected: computed,
            found,
        });
    }
    Ok(())
}

/// Validates a YYMMDD MRZ date and returns the expanded ISO form.
///
/// MRZ dates carry two-digit years. `century_base` is the year the caller
/// considers the start of the current century (for example 2000); a year below
/// the corresponding pivot is read as the previous century. Passport dates of
/// birth are in the past and expiry dates are in the future, so the caller
/// passes the appropriate base for the field being parsed rather than this
/// function guessing.
pub fn parse_mrz_date(raw: &str, century_base: u16) -> Result<String, KycError> {
    if raw.len() != 6 || !raw.bytes().all(|b| b.is_ascii_digit()) {
        return Err(KycError::MrzDateMalformed { field: raw.to_string() });
    }
    let yy: u16 = raw[0..2].parse().map_err(|_| KycError::MrzDateMalformed { field: raw.to_string() })?;
    let mm: u32 = raw[2..4].parse().map_err(|_| KycError::MrzDateMalformed { field: raw.to_string() })?;
    let dd: u32 = raw[4..6].parse().map_err(|_| KycError::MrzDateMalformed { field: raw.to_string() })?;

    if !(1..=12).contains(&mm) {
        return Err(KycError::MrzDateMalformed { field: raw.to_string() });
    }
    if !(1..=31).contains(&dd) {
        return Err(KycError::MrzDateMalformed { field: raw.to_string() });
    }

    let pivot = century_base % 100;
    let year = if yy < pivot {
        century_base + 100 + yy
    } else {
        century_base + yy
    };

    // Reject calendar-impossible dates such as 31 February.
    if dd > days_in_month(year, mm) {
        return Err(KycError::MrzDateMalformed { field: raw.to_string() });
    }

    Ok(format!("{year:04}-{mm:02}-{dd:02}"))
}

fn days_in_month(year: u16, month: u32) -> u32 {
    match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        2 => {
            let leap = (year % 4 == 0 && year % 100 != 0) || year % 400 == 0;
            if leap {
                29
            } else {
                28
            }
        }
        _ => 0,
    }
}

/// Strips MRZ filler characters and converts `<` field separators to spaces.
fn clean_field(s: &str) -> String {
    s.replace('<', " ").trim().to_string()
}

/// Splits an MRZ name field (`SURNAME<<GIVEN<NAMES`) into its two parts.
fn split_names(field: &str) -> (String, String) {
    let mut parts = field.splitn(2, "<<");
    let surname = clean_field(parts.next().unwrap_or(""));
    let given = clean_field(parts.next().unwrap_or(""));
    (surname, given)
}

/// Parses and fully validates a machine-readable zone.
///
/// The input is the raw MRZ text; lines are split on newlines or carriage
/// returns, blank lines are ignored, and each line is padded with filler
/// characters to its expected length so that a document printed without
/// trailing fillers still validates.
pub fn parse(input: &str) -> Result<Mrz, KycError> {
    let mut lines: Vec<String> = input
        .lines()
        .map(|l| l.trim_end_matches(['\r', ' ']).to_string())
        .filter(|l| !l.is_empty())
        .collect();

    if lines.is_empty() {
        return Err(KycError::MrzEmpty);
    }

    // Document type is decided by the line count and length, which is how the
    // formats are distinguished in practice.
    match lines.len() {
        2 => {
            for l in lines.iter_mut() {
                if l.len() > TD3_LINE_LEN {
                    return Err(KycError::MrzLineTooLong {
                        found: l.len(),
                        max: TD3_LINE_LEN,
                    });
                }
                while l.len() < TD3_LINE_LEN {
                    l.push('<');
                }
            }
            parse_td3(&lines[0], &lines[1])
        }
        3 => {
            for l in lines.iter_mut() {
                if l.len() > TD1_LINE_LEN {
                    return Err(KycError::MrzLineTooLong {
                        found: l.len(),
                        max: TD1_LINE_LEN,
                    });
                }
                while l.len() < TD1_LINE_LEN {
                    l.push('<');
                }
            }
            parse_td1(&lines[0], &lines[1], &lines[2])
        }
        n => Err(KycError::MrzUnexpectedLineCount { found: n }),
    }
}

/// Parses a TD3 (passport) MRZ.
///
/// Line 1: `PPPPNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNNN`
///   type(2) country(3) name(39)
/// Line 2:
///   number(9) check(1) nationality(3) birth(6) check(1) sex(1) expiry(6)
///   check(1) personal(14) check(1) composite(1)
fn parse_td3(line1: &str, line2: &str) -> Result<Mrz, KycError> {
    let b1 = line1.as_bytes();
    let b2 = line2.as_bytes();
    if b1.len() != TD3_LINE_LEN || b2.len() != TD3_LINE_LEN {
        return Err(KycError::MrzWrongLength {
            format: MrzFormat::Td3,
            line1: b1.len(),
            line2: b2.len(),
        });
    }

    let document_type = String::from_utf8_lossy(&b1[0..2]).to_string();
    let issuing_country = String::from_utf8_lossy(&b1[2..5]).to_string();
    let name_field = String::from_utf8_lossy(&b1[5..44]).to_string();

    let document_number = String::from_utf8_lossy(&b2[0..9]).to_string();
    let number_check = b2[9];
    let nationality = String::from_utf8_lossy(&b2[10..13]).to_string();
    let dob = String::from_utf8_lossy(&b2[13..19]).to_string();
    let dob_check = b2[19];
    let sex_char = b2[20];
    let expiry = String::from_utf8_lossy(&b2[21..27]).to_string();
    let expiry_check = b2[27];
    let personal = String::from_utf8_lossy(&b2[28..42]).to_string();
    let personal_check = b2[42];
    let composite_check = b2[43];

    // Document number: the number field may contain fillers, which count as 0.
    verify_check_digit(&document_number, number_check)?;
    verify_check_digit(&dob, dob_check)?;
    verify_check_digit(&expiry, expiry_check)?;
    verify_check_digit(&personal, personal_check)?;

    // The composite check digit covers the document number, its check digit,
    // the birth date, its check digit, the expiry date, its check digit, the
    // personal number and its check digit.
    let composite_input = format!(
        "{document_number}{}{dob}{}{expiry}{}{personal}{}",
        number_check as char, dob_check as char, expiry_check as char, personal_check as char
    );
    verify_check_digit(&composite_input, composite_check)?;

    let (surname, given_names) = split_names(&name_field);

    Ok(Mrz {
        format: MrzFormat::Td3,
        document_type,
        issuing_country,
        document_number: document_number.replace('<', ""),
        nationality,
        surname,
        given_names,
        date_of_birth: dob,
        sex: parse_sex(sex_char),
        date_of_expiry: expiry,
        personal_number: if personal.replace('<', "").is_empty() {
            None
        } else {
            Some(personal.replace('<', ""))
        },
    })
}

/// Parses a TD1 (identity card) MRZ.
///
/// Line 1: document code(2) country(3) number(9) check(1) optional(15)
/// Line 2: birth(6) check(1) sex(1) expiry(6) check(1) nationality(3) optional(11) composite(1)
/// Line 3: surname(30) given names(30)
fn parse_td1(line1: &str, line2: &str, line3: &str) -> Result<Mrz, KycError> {
    let b1 = line1.as_bytes();
    let b2 = line2.as_bytes();
    let b3 = line3.as_bytes();
    if b1.len() != TD1_LINE_LEN || b2.len() != TD1_LINE_LEN || b3.len() != TD1_LINE_LEN {
        return Err(KycError::MrzWrongLength {
            format: MrzFormat::Td1,
            line1: b1.len(),
            line2: b2.len(),
        });
    }

    let document_type = String::from_utf8_lossy(&b1[0..2]).to_string();
    let issuing_country = String::from_utf8_lossy(&b1[2..5]).to_string();
    let document_number = String::from_utf8_lossy(&b1[5..14]).to_string();
    let number_check = b1[14];
    let optional1 = String::from_utf8_lossy(&b1[15..30]).to_string();

    let dob = String::from_utf8_lossy(&b2[0..6]).to_string();
    let dob_check = b2[6];
    let sex_char = b2[7];
    let expiry = String::from_utf8_lossy(&b2[8..14]).to_string();
    let expiry_check = b2[14];
    let nationality = String::from_utf8_lossy(&b2[15..18]).to_string();
    let optional2 = String::from_utf8_lossy(&b2[18..29]).to_string();
    let composite_check = b2[29];

    let name_field = String::from_utf8_lossy(b3).to_string();

    verify_check_digit(&document_number, number_check)?;
    verify_check_digit(&dob, dob_check)?;
    verify_check_digit(&expiry, expiry_check)?;

    // TD1 composite: number + its check, optional1, birth + its check, expiry
    // + its check, optional2.
    let composite_input = format!(
        "{document_number}{}{optional1}{dob}{}{expiry}{}{optional2}",
        number_check as char, dob_check as char, expiry_check as char
    );
    verify_check_digit(&composite_input, composite_check)?;

    let (surname, given_names) = split_names(&name_field);

    Ok(Mrz {
        format: MrzFormat::Td1,
        document_type,
        issuing_country,
        document_number: document_number.replace('<', ""),
        nationality,
        surname,
        given_names,
        date_of_birth: dob,
        sex: parse_sex(sex_char),
        date_of_expiry: expiry,
        personal_number: {
            let o = optional2.replace('<', "");
            if o.is_empty() {
                None
            } else {
                Some(o)
            }
        },
    })
}

fn parse_sex(c: u8) -> Sex {
    match c {
        b'M' => Sex::Male,
        b'F' => Sex::Female,
        _ => Sex::Unspecified,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn check_digit_matches_icao_worked_example() {
        // ICAO 9303 part 3 appendix B worked example.
        // "L898902C3" has check digit 6.
        assert_eq!(check_digit("L898902C3").unwrap(), 6);
    }

    #[test]
    fn check_digit_of_all_fillers_is_zero() {
        assert_eq!(check_digit("<<<<<<").unwrap(), 0);
    }

    #[test]
    fn check_digit_of_known_birth_date() {
        // 740812 has check digit 2 (ICAO worked example).
        assert_eq!(check_digit("740812").unwrap(), 2);
    }

    #[test]
    fn check_digit_of_known_expiry_date() {
        // 120415 has check digit 9 (ICAO worked example).
        assert_eq!(check_digit("120415").unwrap(), 9);
    }

    #[test]
    fn check_digit_rejects_lowercase() {
        assert!(check_digit("abc").is_err());
    }

    #[test]
    fn verify_check_digit_accepts_correct_and_rejects_wrong() {
        assert!(verify_check_digit("L898902C3", b'6').is_ok());
        assert!(verify_check_digit("L898902C3", b'7').is_err());
    }

    #[test]
    fn verify_check_digit_rejects_non_numeric() {
        let err = verify_check_digit("L898902C3", b'X').unwrap_err();
        assert!(matches!(err, KycError::MrzCheckDigitNotNumeric { .. }));
    }

    #[test]
    fn parse_mrz_date_expands_century() {
        assert_eq!(parse_mrz_date("740812", 1900).unwrap(), "1974-08-12");
        assert_eq!(parse_mrz_date("120415", 2000).unwrap(), "2012-04-15");
    }

    #[test]
    fn parse_mrz_date_rejects_impossible_dates() {
        assert!(parse_mrz_date("740231", 1900).is_err(), "31 February must be rejected");
        assert!(parse_mrz_date("741312", 1900).is_err(), "month 13 must be rejected");
        assert!(parse_mrz_date("74AB12", 1900).is_err(), "non-numeric must be rejected");
    }

    #[test]
    fn parse_mrz_date_handles_leap_years() {
        assert!(parse_mrz_date("000229", 2000).is_ok(), "2000 is a leap year");
        assert!(parse_mrz_date("010229", 2000).is_err(), "2001 is not a leap year");
        assert!(parse_mrz_date("040229", 2000).is_ok(), "2004 is a leap year");
        assert!(parse_mrz_date("000229", 1900).is_err(), "1900 is not a leap year");
    }

    #[test]
    fn split_names_handles_multi_part_names() {
        let (surname, given) = split_names("MUSTERMANN<<ERIKA<MARIA");
        assert_eq!(surname, "MUSTERMANN");
        assert_eq!(given, "ERIKA MARIA");
    }

    #[test]
    fn split_names_handles_surname_only() {
        let (surname, given) = split_names("CHER<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<");
        assert_eq!(surname, "CHER");
        assert_eq!(given, "");
    }

    #[test]
    fn parse_rejects_empty_input() {
        assert!(matches!(parse("").unwrap_err(), KycError::MrzEmpty));
    }

    #[test]
    fn parse_rejects_wrong_line_count() {
        let err = parse("ONE LINE ONLY").unwrap_err();
        assert!(matches!(err, KycError::MrzUnexpectedLineCount { found: 1 }));
    }

    #[test]
    fn parse_rejects_overlong_line() {
        let long = "P".repeat(50);
        let err = parse(&format!("{long}\n{long}")).unwrap_err();
        assert!(matches!(err, KycError::MrzLineTooLong { .. }));
    }

    /// The canonical ICAO 9303 TD3 specimen, which must fully validate.
    #[test]
    fn parses_icao_td3_specimen() {
        let mrz = "P<UTOERIKSSON<<ANNA<MARIA<<<<<<<<<<<<<<<<<<<\n\
                   L898902C36UTO7408122F1204159ZE184226B<<<<<10";
        let parsed = parse(mrz).expect("the ICAO specimen must validate");
        assert_eq!(parsed.format, MrzFormat::Td3);
        assert_eq!(parsed.document_type, "P<");
        assert_eq!(parsed.issuing_country, "UTO");
        assert_eq!(parsed.document_number, "L898902C3");
        assert_eq!(parsed.nationality, "UTO");
        assert_eq!(parsed.surname, "ERIKSSON");
        assert_eq!(parsed.given_names, "ANNA MARIA");
        assert_eq!(parsed.sex, Sex::Female);
        assert_eq!(parsed.date_of_birth, "740812");
        assert_eq!(parsed.date_of_expiry, "120415");
    }

    /// A single corrupted character must be caught by the composite check
    /// digit, which is the whole point of the checksum.
    #[test]
    fn rejects_td3_with_corrupted_document_number() {
        // Change the document number without recomputing its check digit.
        let mrz = "P<UTOERIKSSON<<ANNA<MARIA<<<<<<<<<<<<<<<<<<<\n\
                   L898902X36UTO7408122F1204159ZE184226B<<<<<10";
        let err = parse(mrz).unwrap_err();
        assert!(
            matches!(err, KycError::MrzCheckDigitMismatch { .. }),
            "a corrupted number must fail a check digit, got {err:?}"
        );
    }

    /// Corrupting only the composite digit must also be caught, proving the
    /// composite is verified and not merely computed.
    #[test]
    fn rejects_td3_with_corrupted_composite_check_digit() {
        let mrz = "P<UTOERIKSSON<<ANNA<MARIA<<<<<<<<<<<<<<<<<<<\n\
                   L898902C36UTO7408122F1204159ZE184226B<<<<<11";
        let err = parse(mrz).unwrap_err();
        assert!(
            matches!(err, KycError::MrzCheckDigitMismatch { .. }),
            "a corrupted composite must fail, got {err:?}"
        );
    }

    #[test]
    fn parses_icao_td1_specimen() {
        // ICAO 9303 part 5 TD1 specimen (identity card).
        let mrz = "I<UTOD231458907<<<<<<<<<<<<<<<\n\
                   7408122F1204159UTO<<<<<<<<<<<6\n\
                   ERIKSSON<<ANNA<MARIA<<<<<<<<<<";
        let parsed = parse(mrz).expect("the ICAO TD1 specimen must validate");
        assert_eq!(parsed.format, MrzFormat::Td1);
        assert_eq!(parsed.document_number, "D23145890");
        assert_eq!(parsed.surname, "ERIKSSON");
        assert_eq!(parsed.given_names, "ANNA MARIA");
        assert_eq!(parsed.sex, Sex::Female);
    }

    #[test]
    fn pads_short_lines_with_fillers() {
        // A document printed without trailing fillers must still validate.
        let mrz = "P<UTOERIKSSON<<ANNA<MARIA\n\
                   L898902C36UTO7408122F1204159ZE184226B<<<<<10";
        let parsed = parse(mrz).expect("short trailing fillers must be tolerated");
        assert_eq!(parsed.document_number, "L898902C3");
    }

    #[test]
    fn normalised_name_orders_given_then_surname() {
        let mrz = "P<UTOERIKSSON<<ANNA<MARIA<<<<<<<<<<<<<<<<<<<\n\
                   L898902C36UTO7408122F1204159ZE184226B<<<<<10";
        let parsed = parse(mrz).unwrap();
        assert_eq!(parsed.normalised_name(), "ANNA MARIA ERIKSSON");
    }
}
